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
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/components/tenant"
)

const tenantFinalizer = "supabase.codanael.io/tenant-finalizer"

// SupabaseTenantReconciler reconciles a SupabaseTenant object
type SupabaseTenantReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=supabase.codanael.io,resources=supabasetenants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=supabase.codanael.io,resources=supabasetenants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=supabase.codanael.io,resources=supabasetenants/finalizers,verbs=update
//+kubebuilder:rbac:groups=supabase.codanael.io,resources=supabases,verbs=get;list;watch
//+kubebuilder:rbac:groups=postgresql.cnpg.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces;services;secrets;configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *SupabaseTenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Get the SupabaseTenant
	tenantCR := &v1alpha1.SupabaseTenant{}
	if err := r.Get(ctx, req.NamespacedName, tenantCR); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling SupabaseTenant", "tenant", tenantCR.Name, "tenantId", tenantCR.Spec.TenantID)

	// 2. Handle deletion
	if !tenantCR.DeletionTimestamp.IsZero() {
		return r.finalize(ctx, tenantCR)
	}

	// 3. Ensure finalizer
	if !controllerutil.ContainsFinalizer(tenantCR, tenantFinalizer) {
		controllerutil.AddFinalizer(tenantCR, tenantFinalizer)
		if err := r.Update(ctx, tenantCR); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. Resolve parent Supabase
	supabase := &v1alpha1.Supabase{}
	supabaseKey := client.ObjectKey{
		Namespace: tenantCR.Namespace,
		Name:      tenantCR.Spec.SupabaseRef,
	}
	if err := r.Get(ctx, supabaseKey, supabase); err != nil {
		log.Error(err, "Failed to resolve supabaseRef", "ref", tenantCR.Spec.SupabaseRef)
		tenantCR.Status.Phase = v1alpha1.TenantPhaseError
		meta.SetStatusCondition(&tenantCR.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "SupabaseRefNotFound",
			Message:            fmt.Sprintf("Cannot resolve supabaseRef %q: %v", tenantCR.Spec.SupabaseRef, err),
			ObservedGeneration: tenantCR.Generation,
		})
		_ = r.Status().Update(ctx, tenantCR)
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}

	// 5. Create TenantContext
	tctx := components.NewTenantContext(r.Client, r.Scheme, r.Recorder, tenantCR, supabase)

	// 6. Phase 1 — Sequential prerequisites: Namespace → Secret → Database
	prereqComponents := []struct {
		comp      components.Component
		condition string
	}{
		{tenant.NewNamespaceComponent(tctx), v1alpha1.TenantConditionNamespaceReady},
		{tenant.NewSecretComponent(tctx), v1alpha1.TenantConditionSecretsReady},
		{tenant.NewDatabaseComponent(tctx), v1alpha1.TenantConditionDatabaseReady},
	}

	for _, pc := range prereqComponents {
		ready, hasErr := r.reconcileTenantComponent(ctx, tenantCR, pc.comp, pc.condition)
		if hasErr || !ready {
			tenantCR.Status.Phase = v1alpha1.TenantPhaseProvisioning
			tenantCR.Status.ObservedGeneration = tenantCR.Generation
			if err := r.Status().Update(ctx, tenantCR); err != nil {
				log.Error(err, "Failed to update tenant status")
			}
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	// 7. Phase 2 — Service components (continue on individual failures)
	serviceComponents := []struct {
		comp      components.Component
		condition string
	}{
		{tenant.NewAuth(tctx), v1alpha1.TenantConditionAuthReady},
		{tenant.NewREST(tctx), v1alpha1.TenantConditionRESTReady},
		{tenant.NewRealtime(tctx), v1alpha1.TenantConditionRealtimeReady},
		{tenant.NewStorage(tctx), v1alpha1.TenantConditionStorageReady},
		{tenant.NewFunctions(tctx), v1alpha1.TenantConditionFunctionsReady},
	}

	allServicesReady := true
	for _, sc := range serviceComponents {
		ready, _ := r.reconcileTenantComponent(ctx, tenantCR, sc.comp, sc.condition)
		if !ready {
			allServicesReady = false
		}
	}

	// 8. Phase 3 — Routing
	routingComp := tenant.NewRoutingComponent(tctx)
	routingReady, _ := r.reconcileTenantComponent(ctx, tenantCR, routingComp, v1alpha1.TenantConditionRoutingReady)

	// 9. Derive phase
	allReady := allServicesReady && routingReady
	phase := v1alpha1.TenantPhaseProvisioning
	if tenantCR.Spec.Suspended {
		phase = v1alpha1.TenantPhaseSuspended
	} else if allReady {
		phase = v1alpha1.TenantPhaseReady
	}

	// 10. Update status
	tenantCR.Status.Phase = phase
	tenantCR.Status.Namespace = tctx.TenantNamespace
	tenantCR.Status.Endpoint = fmt.Sprintf("%s.%s", tenantCR.Spec.TenantID, supabase.Spec.Gateway.BaseDomain)
	tenantCR.Status.ObservedGeneration = tenantCR.Generation

	if err := r.Status().Update(ctx, tenantCR); err != nil {
		log.Error(err, "Failed to update tenant status")
		return ctrl.Result{}, err
	}

	// 11. Requeue interval
	if phase == v1alpha1.TenantPhaseReady {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}
	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// reconcileTenantComponent reconciles a single tenant component and updates its condition.
// Returns (ready, hasError).
func (r *SupabaseTenantReconciler) reconcileTenantComponent(
	ctx context.Context,
	tenantCR *v1alpha1.SupabaseTenant,
	comp components.Component,
	condType string,
) (bool, bool) {
	log := logf.FromContext(ctx)

	result, err := comp.Reconcile(ctx)
	if err != nil {
		log.Error(err, "Failed to reconcile tenant component", "component", comp.Name())
		meta.SetStatusCondition(&tenantCR.Status.Conditions, metav1.Condition{
			Type:               condType,
			Status:             metav1.ConditionFalse,
			Reason:             "ReconcileError",
			Message:            err.Error(),
			ObservedGeneration: tenantCR.Generation,
		})
		return false, true
	}

	if result.RequeueAfter > 0 {
		meta.SetStatusCondition(&tenantCR.Status.Conditions, metav1.Condition{
			Type:               condType,
			Status:             metav1.ConditionFalse,
			Reason:             "Reconciling",
			Message:            "Component is being reconciled",
			ObservedGeneration: tenantCR.Generation,
		})
		return false, false
	}

	ready, msg, err := comp.Healthcheck(ctx)
	if err != nil {
		log.Error(err, "Failed to healthcheck tenant component", "component", comp.Name())
		meta.SetStatusCondition(&tenantCR.Status.Conditions, metav1.Condition{
			Type:               condType,
			Status:             metav1.ConditionFalse,
			Reason:             "HealthcheckError",
			Message:            err.Error(),
			ObservedGeneration: tenantCR.Generation,
		})
		return false, true
	}

	if ready {
		meta.SetStatusCondition(&tenantCR.Status.Conditions, metav1.Condition{
			Type:               condType,
			Status:             metav1.ConditionTrue,
			Reason:             "Ready",
			Message:            msg,
			ObservedGeneration: tenantCR.Generation,
		})
		return true, false
	}

	meta.SetStatusCondition(&tenantCR.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             metav1.ConditionFalse,
		Reason:             "NotReady",
		Message:            msg,
		ObservedGeneration: tenantCR.Generation,
	})
	return false, false
}

// finalize handles tenant deletion.
func (r *SupabaseTenantReconciler) finalize(ctx context.Context, tenantCR *v1alpha1.SupabaseTenant) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(tenantCR, tenantFinalizer) {
		return ctrl.Result{}, nil
	}

	// Set phase to Deleting
	tenantCR.Status.Phase = v1alpha1.TenantPhaseDeleting
	if err := r.Status().Update(ctx, tenantCR); err != nil {
		log.Error(err, "Failed to set Deleting phase")
	}

	// Resolve supabaseRef (if fails, remove finalizer anyway)
	supabase := &v1alpha1.Supabase{}
	supabaseKey := client.ObjectKey{
		Namespace: tenantCR.Namespace,
		Name:      tenantCR.Spec.SupabaseRef,
	}
	if err := r.Get(ctx, supabaseKey, supabase); err != nil {
		log.Error(err, "Failed to resolve supabaseRef during finalization, removing finalizer anyway")
		controllerutil.RemoveFinalizer(tenantCR, tenantFinalizer)
		return ctrl.Result{}, r.Update(ctx, tenantCR)
	}

	// Create TenantContext
	tctx := components.NewTenantContext(r.Client, r.Scheme, r.Recorder, tenantCR, supabase)

	// Finalize in reverse order: Routing → Database → Namespace
	finalizers := []components.Component{
		tenant.NewRoutingComponent(tctx),
		tenant.NewDatabaseComponent(tctx),
		tenant.NewNamespaceComponent(tctx),
	}

	for _, comp := range finalizers {
		if err := comp.Finalize(ctx); err != nil {
			log.Error(err, "Failed to finalize component", "component", comp.Name())
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(tenantCR, tenantFinalizer)
	return ctrl.Result{}, r.Update(ctx, tenantCR)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SupabaseTenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SupabaseTenant{}).
		Watches(&appsv1.Deployment{}, handler.EnqueueRequestsFromMapFunc(r.mapByTenantLabel)).
		Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(r.mapByTenantLabel)).
		Watches(&cnpgv1.Database{}, handler.EnqueueRequestsFromMapFunc(r.mapByTenantLabel)).
		Watches(&gatewayv1.HTTPRoute{}, handler.EnqueueRequestsFromMapFunc(r.mapByTenantLabel)).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}

func (r *SupabaseTenantReconciler) mapByTenantLabel(ctx context.Context, obj client.Object) []ctrl.Request {
	tenantID := obj.GetLabels()["supabase.codanael.io/tenant"]
	if tenantID == "" {
		return nil
	}
	list := &v1alpha1.SupabaseTenantList{}
	if err := r.List(ctx, list); err != nil {
		return nil
	}
	for _, t := range list.Items {
		if t.Spec.TenantID == tenantID {
			return []ctrl.Request{{NamespacedName: client.ObjectKeyFromObject(&t)}}
		}
	}
	return nil
}
