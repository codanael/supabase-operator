package tenant

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/credentials"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentREST    = "rest"
	defaultRESTImage = "postgrest/postgrest:v14.6"
	restPort         = 3000
)

// RESTBuilder builds the PostgREST deployment and service.
type RESTBuilder struct {
	ctx *components.TenantContext
}

func NewRESTBuilder(ctx *components.TenantContext) *RESTBuilder {
	return &RESTBuilder{ctx: ctx}
}

func (b *RESTBuilder) ComponentName() string {
	return componentREST
}

func (b *RESTBuilder) resourceName() string {
	return fmt.Sprintf("%s-rest", b.ctx.TenantID())
}

func (b *RESTBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentREST)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentREST)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	preset := resources.GetPreset(b.ctx.Tenant.Spec.Resources)
	restSpec := b.ctx.Tenant.Spec.REST

	schemas := "public,graphql_public"
	if len(restSpec.Schemas) > 0 {
		schemas = strings.Join(restSpec.Schemas, ",")
	}

	maxRows := "1000"
	if restSpec.MaxRows != nil {
		maxRows = strconv.Itoa(int(*restSpec.MaxRows))
	}

	env := []corev1.EnvVar{
		{Name: "PGRST_DB_URI", Value: b.ctx.DatabaseDSN("authenticator", b.ctx.DatabasePassword)},
		{Name: "PGRST_DB_SCHEMAS", Value: schemas},
		{Name: "PGRST_DB_MAX_ROWS", Value: maxRows},
		{Name: "PGRST_DB_ANON_ROLE", Value: "anon"},
		{Name: "PGRST_JWT_SECRET", Value: b.ctx.JWTSecret},
		{Name: "PGRST_DB_USE_LEGACY_GUCS", Value: "false"},
		{Name: "PGRST_APP_SETTINGS_JWT_SECRET", Value: b.ctx.JWTSecret},
		{Name: "PGRST_APP_SETTINGS_JWT_EXP", Value: "3600"},
	}

	return resources.NewDeploymentBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithReplicas(preset.Replicas).
		WithPodAnnotations(map[string]string{
			credentials.SecretHashAnnotation: b.ctx.SecretHash,
		}).
		WithContainer(corev1.Container{
			Name:  componentREST,
			Image: defaultRESTImage,
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: restPort, Protocol: corev1.ProtocolTCP},
			},
			Env:       env,
			Resources: preset.Resources,
		}).
		Build()
}

func (b *RESTBuilder) BuildService() *corev1.Service {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentREST)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentREST)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	return resources.NewServiceBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("http", restPort, restPort).
		Build()
}

func NewREST(ctx *components.TenantContext) *TenantServiceComponent {
	return NewTenantServiceComponent(ctx, NewRESTBuilder(ctx))
}
