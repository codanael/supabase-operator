/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	supabasev1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/components/platform"
)

// SupabaseReconciler reconciles a Supabase object
type SupabaseReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=supabase.codanael.io,resources=supabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=supabase.codanael.io,resources=supabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=supabase.codanael.io,resources=supabases/finalizers,verbs=update
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=scheduledbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *SupabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Supabase instance
	sb := &supabasev1alpha1.Supabase{}
	if err := r.Get(ctx, req.NamespacedName, sb); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling Supabase", "name", sb.Name)

	// Save a deep copy as patch base before any status modifications
	statusPatch := client.MergeFrom(sb.DeepCopy())

	// Preflight: check that we can list CNPG clusters
	if err := r.preflightChecks(ctx, sb); err != nil {
		return r.updateStatus(ctx, sb, statusPatch, supabasev1alpha1.SupphaseError, err)
	}

	// Build platform context
	pctx := &components.PlatformContext{
		Client:   r.Client,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
		Supabase: sb,
	}

	// Map component names to condition types
	conditionMap := map[string]string{
		"database":  supabasev1alpha1.ConditionDatabaseReady,
		"backup":    supabasev1alpha1.ConditionBackupReady,
		"gateway":   supabasev1alpha1.ConditionGatewayReady,
		"imgproxy":  supabasev1alpha1.ConditionImgproxyReady,
		"analytics": supabasev1alpha1.ConditionAnalyticsReady,
		"vector":    supabasev1alpha1.ConditionVectorReady,
		"supavisor": supabasev1alpha1.ConditionSupavisorReady,
		"studio":    supabasev1alpha1.ConditionStudioReady,
	}

	hasError := false
	allReady := true
	dbReady := false

	// Reconcile critical components first (DB, Gateway).
	// If either fails, skip all downstream service components.
	criticalComponents := []components.Component{
		platform.NewCNPGCluster(pctx),
		platform.NewScheduledBackup(pctx),
		platform.NewGateway(pctx),
	}

	criticalFailed := false
	for _, comp := range criticalComponents {
		condType := conditionMap[comp.Name()]
		ready, compErr := r.reconcileComponent(ctx, sb, comp, condType)
		if compErr {
			hasError = true
			criticalFailed = true
		}
		if !ready {
			allReady = false
		}
		if comp.Name() == "database" {
			dbReady = ready
		}
	}

	// Only reconcile service components if critical ones did not fail
	if !criticalFailed {
		serviceComponents := []components.Component{
			platform.NewImgproxy(pctx),
			platform.NewAnalytics(pctx),
			platform.NewVector(pctx),
			platform.NewSupavisor(pctx),
			platform.NewStudio(pctx),
		}

		for _, comp := range serviceComponents {
			condType := conditionMap[comp.Name()]
			ready, compErr := r.reconcileComponent(ctx, sb, comp, condType)
			if compErr {
				hasError = true
			}
			if !ready {
				allReady = false
			}
		}
	} else {
		// Mark downstream components as unknown since we skipped them
		allReady = false
	}

	// Update status fields
	sb.Status.DatabaseReady = isConditionTrue(sb, supabasev1alpha1.ConditionDatabaseReady)
	sb.Status.GatewayReady = isConditionTrue(sb, supabasev1alpha1.ConditionGatewayReady)

	// Derive phase
	phase := derivePhase(hasError, allReady, dbReady)
	return r.updateStatus(ctx, sb, statusPatch, phase, nil)
}

func (r *SupabaseReconciler) preflightChecks(ctx context.Context, sb *supabasev1alpha1.Supabase) error {
	clusterList := &cnpgv1.ClusterList{}
	if err := r.List(ctx, clusterList, client.InNamespace(sb.Namespace), client.Limit(1)); err != nil {
		return fmt.Errorf("preflight: cannot list CNPG Clusters (is the CRD installed?): %w", err)
	}

	gwList := &gatewayv1.GatewayList{}
	if err := r.List(ctx, gwList, client.InNamespace(sb.Namespace), client.Limit(1)); err != nil {
		return fmt.Errorf("preflight: cannot list Gateways (are Gateway API CRDs installed?): %w", err)
	}

	return nil
}

// reconcileComponent reconciles a single component and updates its condition.
// Returns (ready, hasError).
func (r *SupabaseReconciler) reconcileComponent(ctx context.Context, sb *supabasev1alpha1.Supabase, comp components.Component, condType string) (bool, bool) {
	log := logf.FromContext(ctx)

	result, err := comp.Reconcile(ctx)
	if err != nil {
		log.Error(err, "Failed to reconcile component", "component", comp.Name())
		meta.SetStatusCondition(&sb.Status.Conditions, metav1.Condition{
			Type:               condType,
			Status:             metav1.ConditionFalse,
			Reason:             "ReconcileError",
			Message:            err.Error(),
			ObservedGeneration: sb.Generation,
		})
		return false, true
	}

	if result.RequeueAfter > 0 {
		meta.SetStatusCondition(&sb.Status.Conditions, metav1.Condition{
			Type:               condType,
			Status:             metav1.ConditionFalse,
			Reason:             "Reconciling",
			Message:            "Component is being reconciled",
			ObservedGeneration: sb.Generation,
		})
		return false, false
	}

	ready, msg, err := comp.Healthcheck(ctx)
	if err != nil {
		log.Error(err, "Failed to healthcheck component", "component", comp.Name())
		meta.SetStatusCondition(&sb.Status.Conditions, metav1.Condition{
			Type:               condType,
			Status:             metav1.ConditionFalse,
			Reason:             "HealthcheckError",
			Message:            err.Error(),
			ObservedGeneration: sb.Generation,
		})
		return false, true
	}

	if ready {
		meta.SetStatusCondition(&sb.Status.Conditions, metav1.Condition{
			Type:               condType,
			Status:             metav1.ConditionTrue,
			Reason:             "Ready",
			Message:            msg,
			ObservedGeneration: sb.Generation,
		})
		return true, false
	}

	meta.SetStatusCondition(&sb.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             metav1.ConditionFalse,
		Reason:             "NotReady",
		Message:            msg,
		ObservedGeneration: sb.Generation,
	})
	return false, false
}

func (r *SupabaseReconciler) updateStatus(ctx context.Context, sb *supabasev1alpha1.Supabase, statusPatch client.Patch, phase string, reconcileErr error) (ctrl.Result, error) {
	sb.Status.Phase = phase
	sb.Status.ObservedGeneration = sb.Generation

	if err := r.Status().Patch(ctx, sb, statusPatch); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to patch Supabase status")
		return ctrl.Result{}, err
	}

	switch phase {
	case supabasev1alpha1.SupphaseReady:
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, reconcileErr
	case supabasev1alpha1.SupphaseProvisioning, supabasev1alpha1.SupphaseDegraded:
		return ctrl.Result{RequeueAfter: 30 * time.Second}, reconcileErr
	default:
		return ctrl.Result{}, reconcileErr
	}
}

// derivePhase determines the overall phase from component states.
// dbReady indicates whether the database (critical infrastructure) is healthy.
func derivePhase(hasError, allReady, dbReady bool) string {
	if hasError && !dbReady {
		return supabasev1alpha1.SupphaseError
	}
	if allReady {
		return supabasev1alpha1.SupphaseReady
	}
	// Database is ready but some other components are not
	if dbReady && hasError {
		return supabasev1alpha1.SupphaseDegraded
	}
	return supabasev1alpha1.SupphaseProvisioning
}

func isConditionTrue(sb *supabasev1alpha1.Supabase, condType string) bool {
	return meta.IsStatusConditionTrue(sb.Status.Conditions, condType)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SupabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&supabasev1alpha1.Supabase{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&cnpgv1.Cluster{}).
		Owns(&cnpgv1.ScheduledBackup{}).
		Owns(&gatewayv1.Gateway{}).
		Named("supabase").
		Complete(r)
}
