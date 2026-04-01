package tenant

import (
	"fmt"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentRealtime    = "realtime"
	defaultRealtimeImage = "supabase/realtime:v2.76.5"
	realtimePort         = 4000
)

// RealtimeBuilder builds the Realtime deployment and service.
type RealtimeBuilder struct {
	ctx *components.TenantContext
}

func NewRealtimeBuilder(ctx *components.TenantContext) *RealtimeBuilder {
	return &RealtimeBuilder{ctx: ctx}
}

func (b *RealtimeBuilder) ComponentName() string {
	return componentRealtime
}

func (b *RealtimeBuilder) resourceName() string {
	return fmt.Sprintf("%s-realtime", b.ctx.TenantID())
}

func (b *RealtimeBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentRealtime)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentRealtime)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	preset := resources.GetPreset(b.ctx.Tenant.Spec.Resources)

	env := []corev1.EnvVar{
		{Name: "PORT", Value: fmt.Sprintf("%d", realtimePort)},
		{Name: "DB_HOST", Value: b.ctx.DatabaseHost},
		{Name: "DB_PORT", Value: b.ctx.DatabasePort},
		{Name: "DB_USER", Value: "supabase_admin"},
		{Name: "DB_PASSWORD", Value: b.ctx.DatabasePassword},
		{Name: "DB_NAME", Value: b.ctx.DatabaseName},
		{Name: "DB_AFTER_CONNECT_QUERY", Value: "SET search_path TO _realtime"},
		{Name: "DB_ENC_KEY", Value: "supabaserealtime"},
		{Name: "API_JWT_SECRET", Value: b.ctx.JWTSecret},
		{Name: "SECRET_KEY_BASE", Value: b.ctx.JWTSecret},
		{Name: "ERL_AFLAGS", Value: "-proto_dist inet_tcp"},
		{Name: "DNS_NODES", Value: "''"},
		{Name: "APP_NAME", Value: "realtime"},
		{Name: "SEED_SELF_HOST", Value: "true"},
	}

	return resources.NewDeploymentBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithReplicas(preset.Replicas).
		WithContainer(corev1.Container{
			Name:  componentRealtime,
			Image: defaultRealtimeImage,
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: realtimePort, Protocol: corev1.ProtocolTCP},
			},
			Env:       env,
			Resources: preset.Resources,
		}).
		Build()
}

func (b *RealtimeBuilder) BuildService() *corev1.Service {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentRealtime)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentRealtime)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	return resources.NewServiceBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("http", realtimePort, realtimePort).
		Build()
}

func NewRealtime(ctx *components.TenantContext) *TenantServiceComponent {
	return NewTenantServiceComponent(ctx, NewRealtimeBuilder(ctx))
}
