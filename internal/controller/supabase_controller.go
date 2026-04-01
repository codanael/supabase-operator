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

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *SupabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Supabase instance
	sb := &supabasev1alpha1.Supabase{}
	if err := r.Get(ctx, req.NamespacedName, sb); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling Supabase", "name", sb.Name)

	// Preflight: check that we can list CNPG clusters
	if err := r.preflightChecks(ctx, sb); err != nil {
		return r.updateStatus(ctx, sb, supabasev1alpha1.SupphaseError, err)
	}

	// Build platform context
	pctx := &components.PlatformContext{
		Client:   r.Client,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
		Supabase: sb,
	}

	// Define component order
	componentList := []components.Component{
		platform.NewCNPGCluster(pctx),
		platform.NewGateway(pctx),
		platform.NewImgproxy(pctx),
		platform.NewAnalytics(pctx),
		platform.NewVector(pctx),
		platform.NewSupavisor(pctx),
		platform.NewStudio(pctx),
	}

	// Map component names to condition types
	conditionMap := map[string]string{
		"database":  supabasev1alpha1.ConditionDatabaseReady,
		"gateway":   supabasev1alpha1.ConditionGatewayReady,
		"imgproxy":  supabasev1alpha1.ConditionImgproxyReady,
		"analytics": supabasev1alpha1.ConditionAnalyticsReady,
		"vector":    supabasev1alpha1.ConditionVectorReady,
		"supavisor": supabasev1alpha1.ConditionSupavisorReady,
		"studio":    supabasev1alpha1.ConditionStudioReady,
	}

	hasError := false
	allReady := true

	for _, comp := range componentList {
		condType := conditionMap[comp.Name()]

		// Reconcile component
		result, err := comp.Reconcile(ctx)
		if err != nil {
			log.Error(err, "Failed to reconcile component", "component", comp.Name())
			setCondition(sb, condType, metav1.ConditionFalse, "ReconcileError", err.Error())
			hasError = true
			continue
		}

		if result.Requeue || result.RequeueAfter > 0 {
			// Still processing, mark as not ready but not error
			setCondition(sb, condType, metav1.ConditionFalse, "Reconciling", "Component is being reconciled")
			allReady = false
			continue
		}

		// Healthcheck component
		ready, msg, err := comp.Healthcheck(ctx)
		if err != nil {
			log.Error(err, "Failed to healthcheck component", "component", comp.Name())
			setCondition(sb, condType, metav1.ConditionFalse, "HealthcheckError", err.Error())
			hasError = true
			continue
		}

		if ready {
			setCondition(sb, condType, metav1.ConditionTrue, "Ready", msg)
		} else {
			setCondition(sb, condType, metav1.ConditionFalse, "NotReady", msg)
			allReady = false
		}
	}

	// Update status fields
	sb.Status.DatabaseReady = isConditionTrue(sb, supabasev1alpha1.ConditionDatabaseReady)
	sb.Status.GatewayReady = isConditionTrue(sb, supabasev1alpha1.ConditionGatewayReady)

	// Derive phase
	phase := derivePhase(hasError, allReady)
	return r.updateStatus(ctx, sb, phase, nil)
}

func (r *SupabaseReconciler) preflightChecks(ctx context.Context, sb *supabasev1alpha1.Supabase) error {
	clusterList := &cnpgv1.ClusterList{}
	if err := r.List(ctx, clusterList, client.InNamespace(sb.Namespace), client.Limit(1)); err != nil {
		return fmt.Errorf("preflight: cannot list CNPG Clusters (is the CRD installed?): %w", err)
	}
	return nil
}

func (r *SupabaseReconciler) updateStatus(ctx context.Context, sb *supabasev1alpha1.Supabase, phase string, reconcileErr error) (ctrl.Result, error) {
	sb.Status.Phase = phase
	sb.Status.ObservedGeneration = sb.Generation

	if err := r.Status().Update(ctx, sb); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to update Supabase status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, reconcileErr
}

func derivePhase(hasError, allReady bool) string {
	if hasError {
		return supabasev1alpha1.SupphaseError
	}
	if allReady {
		return supabasev1alpha1.SupphaseReady
	}
	return supabasev1alpha1.SupphaseProvisioning
}

func setCondition(sb *supabasev1alpha1.Supabase, condType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	for i, c := range sb.Status.Conditions {
		if c.Type == condType {
			if c.Status != status {
				sb.Status.Conditions[i].LastTransitionTime = now
			}
			sb.Status.Conditions[i].Status = status
			sb.Status.Conditions[i].Reason = reason
			sb.Status.Conditions[i].Message = message
			sb.Status.Conditions[i].ObservedGeneration = sb.Generation
			return
		}
	}
	sb.Status.Conditions = append(sb.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: sb.Generation,
	})
}

func isConditionTrue(sb *supabasev1alpha1.Supabase, condType string) bool {
	for _, c := range sb.Status.Conditions {
		if c.Type == condType {
			return c.Status == metav1.ConditionTrue
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *SupabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&supabasev1alpha1.Supabase{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&cnpgv1.Cluster{}).
		Owns(&gatewayv1.Gateway{}).
		Named("supabase").
		Complete(r)
}
