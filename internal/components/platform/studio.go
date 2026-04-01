package platform

import (
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentStudio    = "studio"
	defaultStudioImage = "supabase/studio:2026.03.16-sha-5528817"
	studioPort         = 3000
)

type StudioBuilder struct {
	ctx *components.PlatformContext
}

func NewStudioBuilder(ctx *components.PlatformContext) *StudioBuilder {
	return &StudioBuilder{ctx: ctx}
}

func (b *StudioBuilder) ComponentName() string {
	return componentStudio
}

func (b *StudioBuilder) IsEnabled() bool {
	return b.ctx.Supabase.Spec.Studio.IsEnabled()
}

func (b *StudioBuilder) resourceName() string {
	return fmt.Sprintf("%s-studio", b.ctx.InstanceName())
}

func (b *StudioBuilder) image() string {
	if b.ctx.Supabase.Spec.Images.Studio != nil {
		return *b.ctx.Supabase.Spec.Images.Studio
	}
	return defaultStudioImage
}

func (b *StudioBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentStudio)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentStudio)

	return resources.NewDeploymentBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithReplicas(b.ctx.Supabase.Spec.Studio.GetReplicas()).
		WithContainer(corev1.Container{
			Name:  componentStudio,
			Image: b.image(),
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: studioPort, Protocol: corev1.ProtocolTCP},
			},
			Env: []corev1.EnvVar{
				{Name: "HOSTNAME", Value: "::"},
				{Name: "STUDIO_PORT", Value: fmt.Sprintf("%d", studioPort)},
				{Name: "NEXT_PUBLIC_ENABLE_LOGS", Value: "true"},
				{Name: "NEXT_ANALYTICS_BACKEND_PROVIDER", Value: "postgres"},
			},
			Resources: b.ctx.Supabase.Spec.Studio.Resources,
		}).
		Build()
}

func (b *StudioBuilder) BuildService() *corev1.Service {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentStudio)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentStudio)

	return resources.NewServiceBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("http", studioPort, studioPort).
		Build()
}

func NewStudio(ctx *components.PlatformContext) *ServiceComponent {
	return NewServiceComponent(ctx, NewStudioBuilder(ctx))
}
