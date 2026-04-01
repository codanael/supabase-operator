package platform

import (
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentSupavisor       = "supavisor"
	defaultSupavisorImage    = "supabase/supavisor:2.7.4"
	supavisorAPIPort         = 4000
	supavisorSessionPort     = 5432
	supavisorTransactionPort = 6543
)

type SupavisorBuilder struct {
	ctx *components.PlatformContext
}

func NewSupavisorBuilder(ctx *components.PlatformContext) *SupavisorBuilder {
	return &SupavisorBuilder{ctx: ctx}
}

func (b *SupavisorBuilder) ComponentName() string {
	return componentSupavisor
}

func (b *SupavisorBuilder) IsEnabled() bool {
	return b.ctx.Supabase.Spec.Supavisor.IsEnabled()
}

func (b *SupavisorBuilder) resourceName() string {
	return fmt.Sprintf("%s-supavisor", b.ctx.InstanceName())
}

func (b *SupavisorBuilder) image() string {
	if b.ctx.Supabase.Spec.Images.Supavisor != nil {
		return *b.ctx.Supabase.Spec.Images.Supavisor
	}
	return defaultSupavisorImage
}

func (b *SupavisorBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentSupavisor)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentSupavisor)

	return resources.NewDeploymentBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithReplicas(b.ctx.Supabase.Spec.Supavisor.GetReplicas()).
		WithContainer(corev1.Container{
			Name:  componentSupavisor,
			Image: b.image(),
			Ports: []corev1.ContainerPort{
				{Name: "api", ContainerPort: supavisorAPIPort, Protocol: corev1.ProtocolTCP},
				{Name: "session", ContainerPort: supavisorSessionPort, Protocol: corev1.ProtocolTCP},
				{Name: "transaction", ContainerPort: supavisorTransactionPort, Protocol: corev1.ProtocolTCP},
			},
			Env: []corev1.EnvVar{
				{Name: "PORT", Value: fmt.Sprintf("%d", supavisorAPIPort)},
				{Name: "CLUSTER_POSTGRES", Value: "true"},
				{Name: "REGION", Value: "local"},
				{Name: "ERL_AFLAGS", Value: "-proto_dist inet_tcp"},
				{Name: "POOLER_POOL_MODE", Value: "transaction"},
			},
			Resources: b.ctx.Supabase.Spec.Supavisor.Resources,
		}).
		Build()
}

func (b *SupavisorBuilder) BuildService() *corev1.Service {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentSupavisor)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentSupavisor)

	return resources.NewServiceBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("api", supavisorAPIPort, supavisorAPIPort).
		WithPort("session", supavisorSessionPort, supavisorSessionPort).
		WithPort("transaction", supavisorTransactionPort, supavisorTransactionPort).
		Build()
}

func NewSupavisor(ctx *components.PlatformContext) *ServiceComponent {
	return NewServiceComponent(ctx, NewSupavisorBuilder(ctx))
}
