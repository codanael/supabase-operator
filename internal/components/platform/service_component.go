package platform

import (
	"context"
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ServiceComponentBuilder defines the interface for building Deployment + Service pairs.
type ServiceComponentBuilder interface {
	ComponentName() string
	BuildDeployment() *appsv1.Deployment
	BuildService() *corev1.Service
	IsEnabled() bool
}

// ServiceComponent wraps a ServiceComponentBuilder and implements the Component interface.
type ServiceComponent struct {
	ctx     *components.PlatformContext
	builder ServiceComponentBuilder
}

func NewServiceComponent(ctx *components.PlatformContext, builder ServiceComponentBuilder) *ServiceComponent {
	return &ServiceComponent{ctx: ctx, builder: builder}
}

func (s *ServiceComponent) Name() string {
	return s.builder.ComponentName()
}

func (s *ServiceComponent) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if !s.builder.IsEnabled() {
		return ctrl.Result{}, nil
	}

	// Reconcile Deployment
	desiredDeploy := s.builder.BuildDeployment()
	if err := controllerutil.SetControllerReference(s.ctx.Supabase, desiredDeploy, s.ctx.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("setting owner reference on deployment: %w", err)
	}

	existingDeploy := &appsv1.Deployment{}
	err := s.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desiredDeploy), existingDeploy)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting deployment: %w", err)
	}

	if err != nil {
		if createErr := s.ctx.Client.Create(ctx, desiredDeploy); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating deployment: %w", createErr)
		}
		s.ctx.Recorder.Eventf(s.ctx.Supabase, "Normal", "Created", "Created Deployment %s", desiredDeploy.Name)
	} else {
		existingDeploy.Spec = desiredDeploy.Spec
		existingDeploy.Labels = desiredDeploy.Labels
		if updateErr := s.ctx.Client.Update(ctx, existingDeploy); updateErr != nil {
			return ctrl.Result{}, fmt.Errorf("updating deployment: %w", updateErr)
		}
	}

	// Reconcile Service
	desiredSvc := s.builder.BuildService()
	if err := controllerutil.SetControllerReference(s.ctx.Supabase, desiredSvc, s.ctx.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("setting owner reference on service: %w", err)
	}

	existingSvc := &corev1.Service{}
	err = s.ctx.Client.Get(ctx, client.ObjectKeyFromObject(desiredSvc), existingSvc)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("getting service: %w", err)
	}

	if err != nil {
		if createErr := s.ctx.Client.Create(ctx, desiredSvc); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating service: %w", createErr)
		}
		s.ctx.Recorder.Eventf(s.ctx.Supabase, "Normal", "Created", "Created Service %s", desiredSvc.Name)
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

func (s *ServiceComponent) Healthcheck(ctx context.Context) (bool, string, error) {
	if !s.builder.IsEnabled() {
		return true, "component disabled", nil
	}

	deploy := &appsv1.Deployment{}
	key := client.ObjectKeyFromObject(s.builder.BuildDeployment())
	if err := s.ctx.Client.Get(ctx, key, deploy); err != nil {
		return false, "Deployment not found", client.IgnoreNotFound(err)
	}

	if deploy.Status.ReadyReplicas >= 1 && deploy.Status.ReadyReplicas == *deploy.Spec.Replicas {
		return true, fmt.Sprintf("%d/%d replicas ready", deploy.Status.ReadyReplicas, *deploy.Spec.Replicas), nil
	}

	return false, fmt.Sprintf("%d/%d replicas ready", deploy.Status.ReadyReplicas, *deploy.Spec.Replicas), nil
}

func (s *ServiceComponent) Finalize(ctx context.Context) error {
	return nil
}
