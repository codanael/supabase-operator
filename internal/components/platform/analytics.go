package platform

import (
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentAnalytics    = "analytics"
	defaultAnalyticsImage = "supabase/logflare:1.31.2"
	analyticsPort         = 4000
)

type AnalyticsBuilder struct {
	ctx *components.PlatformContext
}

func NewAnalyticsBuilder(ctx *components.PlatformContext) *AnalyticsBuilder {
	return &AnalyticsBuilder{ctx: ctx}
}

func (b *AnalyticsBuilder) ComponentName() string {
	return componentAnalytics
}

func (b *AnalyticsBuilder) IsEnabled() bool {
	return b.ctx.Supabase.Spec.Analytics.IsEnabled()
}

func (b *AnalyticsBuilder) resourceName() string {
	return fmt.Sprintf("%s-analytics", b.ctx.InstanceName())
}

func (b *AnalyticsBuilder) image() string {
	if b.ctx.Supabase.Spec.Images.Analytics != nil {
		return *b.ctx.Supabase.Spec.Images.Analytics
	}
	return defaultAnalyticsImage
}

func (b *AnalyticsBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentAnalytics)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentAnalytics)

	return resources.NewDeploymentBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithReplicas(b.ctx.Supabase.Spec.Analytics.GetReplicas()).
		WithContainer(corev1.Container{
			Name:  componentAnalytics,
			Image: b.image(),
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: analyticsPort, Protocol: corev1.ProtocolTCP},
			},
			Env: []corev1.EnvVar{
				{Name: "LOGFLARE_NODE_HOST", Value: "127.0.0.1"},
				{Name: "LOGFLARE_SINGLE_TENANT", Value: "true"},
				{Name: "LOGFLARE_SUPABASE_MODE", Value: "true"},
				{Name: "LOGFLARE_FEATURE_FLAG_OVERRIDE", Value: "multibackend=true"},
				{Name: "DB_SCHEMA", Value: "_analytics"},
				{Name: "POSTGRES_BACKEND_SCHEMA", Value: "_analytics"},
			},
			Resources: b.ctx.Supabase.Spec.Analytics.Resources,
		}).
		Build()
}

func (b *AnalyticsBuilder) BuildService() *corev1.Service {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentAnalytics)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentAnalytics)

	return resources.NewServiceBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("http", analyticsPort, analyticsPort).
		Build()
}

func NewAnalytics(ctx *components.PlatformContext) *ServiceComponent {
	return NewServiceComponent(ctx, NewAnalyticsBuilder(ctx))
}
