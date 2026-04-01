package tenant

import (
	"context"
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TenantServiceBuilder defines the interface for building tenant Deployment + Service pairs.
type TenantServiceBuilder interface {
	ComponentName() string
	BuildDeployment() *appsv1.Deployment
	BuildService() *corev1.Service
}

// TenantServiceComponent wraps a TenantServiceBuilder and implements the Component interface.
type TenantServiceComponent struct {
	ctx     *components.TenantContext
	builder TenantServiceBuilder
}

func NewTenantServiceComponent(ctx *components.TenantContext, builder TenantServiceBuilder) *TenantServiceComponent {
	return &TenantServiceComponent{ctx: ctx, builder: builder}
}

func (s *TenantServiceComponent) Name() string {
	return s.builder.ComponentName()
}

func (s *TenantServiceComponent) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if s.ctx.Tenant.Spec.Suspended {
		// Scale to 0 by updating existing deployment replicas
		deploy := &appsv1.Deployment{}
		key := client.ObjectKeyFromObject(s.builder.BuildDeployment())
		err := s.ctx.Client.Get(ctx, key, deploy)
		if err == nil {
			zero := int32(0)
			deploy.Spec.Replicas = &zero
			if updateErr := s.ctx.Client.Update(ctx, deploy); updateErr != nil {
				return ctrl.Result{}, fmt.Errorf("scaling deployment to zero: %w", updateErr)
			}
		}
		return ctrl.Result{}, nil
	}

	// Reconcile Deployment
	desiredDeploy := s.builder.BuildDeployment()
	existingDeploy := &appsv1.Deployment{}
	err := s.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desiredDeploy), existingDeploy)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting deployment: %w", err)
	}

	if err != nil {
		if createErr := s.ctx.Client.Create(ctx, desiredDeploy); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating deployment: %w", createErr)
		}
		s.ctx.Recorder.Eventf(s.ctx.Tenant, "Normal", "Created", "Created Deployment %s", desiredDeploy.Name)
	} else {
		existingDeploy.Spec = desiredDeploy.Spec
		existingDeploy.Labels = desiredDeploy.Labels
		if updateErr := s.ctx.Client.Update(ctx, existingDeploy); updateErr != nil {
			return ctrl.Result{}, fmt.Errorf("updating deployment: %w", updateErr)
		}
	}

	// Reconcile Service
	desiredSvc := s.builder.BuildService()
	existingSvc := &corev1.Service{}
	err = s.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desiredSvc), existingSvc)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting service: %w", err)
	}

	if err != nil {
		if createErr := s.ctx.Client.Create(ctx, desiredSvc); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating service: %w", createErr)
		}
		s.ctx.Recorder.Eventf(s.ctx.Tenant, "Normal", "Created", "Created Service %s", desiredSvc.Name)
	} else {
		existingSvc.Spec.Ports = desiredSvc.Spec.Ports
		existingSvc.Spec.Selector = desiredSvc.Spec.Selector
		existingSvc.Labels = desiredSvc.Labels
		if updateErr := s.ctx.Client.Update(ctx, existingSvc); updateErr != nil {
			return ctrl.Result{}, fmt.Errorf("updating service: %w", updateErr)
		}
	}

	return ctrl.Result{}, nil
}

func (s *TenantServiceComponent) Healthcheck(ctx context.Context) (bool, string, error) {
	if s.ctx.Tenant.Spec.Suspended {
		return true, "suspended", nil
	}

	deploy := &appsv1.Deployment{}
	key := client.ObjectKeyFromObject(s.builder.BuildDeployment())
	if err := s.ctx.Client.Get(ctx, key, deploy); err != nil {
		return false, "Deployment not found", client.IgnoreNotFound(err)
	}

	replicas := int32(1)
	if deploy.Spec.Replicas != nil {
		replicas = *deploy.Spec.Replicas
	}

	if deploy.Status.ReadyReplicas >= 1 && deploy.Status.ReadyReplicas == replicas {
		return true, fmt.Sprintf("%d/%d replicas ready", deploy.Status.ReadyReplicas, replicas), nil
	}

	return false, fmt.Sprintf("%d/%d replicas ready", deploy.Status.ReadyReplicas, replicas), nil
}

func (s *TenantServiceComponent) Finalize(_ context.Context) error {
	// No-op: namespace cascade handles deletion
	return nil
}
