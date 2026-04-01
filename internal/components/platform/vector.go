package platform

import (
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentVector      = "vector"
	defaultVectorImage   = "timberio/vector:0.53.0-alpine"
	vectorPort           = 9001
)

type VectorBuilder struct {
	ctx *components.PlatformContext
}

func NewVectorBuilder(ctx *components.PlatformContext) *VectorBuilder {
	return &VectorBuilder{ctx: ctx}
}

func (b *VectorBuilder) ComponentName() string {
	return componentVector
}

func (b *VectorBuilder) IsEnabled() bool {
	return b.ctx.Supabase.Spec.Vector.IsEnabled()
}

func (b *VectorBuilder) resourceName() string {
	return fmt.Sprintf("%s-vector", b.ctx.InstanceName())
}

func (b *VectorBuilder) image() string {
	if b.ctx.Supabase.Spec.Images.Vector != nil {
		return *b.ctx.Supabase.Spec.Images.Vector
	}
	return defaultVectorImage
}

func (b *VectorBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentVector)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentVector)

	return resources.NewDeploymentBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithReplicas(b.ctx.Supabase.Spec.Vector.GetReplicas()).
		WithContainer(corev1.Container{
			Name:  componentVector,
			Image: b.image(),
			Args:  []string{"--config", "/etc/vector/vector.yml"},
			Ports: []corev1.ContainerPort{
				{Name: "api", ContainerPort: vectorPort, Protocol: corev1.ProtocolTCP},
			},
			Resources: b.ctx.Supabase.Spec.Vector.Resources,
		}).
		Build()
}

func (b *VectorBuilder) BuildService() *corev1.Service {
	labels := resources.PlatformLabels(b.ctx.InstanceName(), componentVector)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentVector)

	return resources.NewServiceBuilder(b.ctx.Namespace(), b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("api", vectorPort, vectorPort).
		Build()
}

func NewVector(ctx *components.PlatformContext) *ServiceComponent {
	return NewServiceComponent(ctx, NewVectorBuilder(ctx))
}
