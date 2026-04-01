package tenant

import (
	"fmt"
	"strconv"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/credentials"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentFunctions    = "functions"
	defaultFunctionsImage = "supabase/edge-runtime:v1.71.2"
	functionsPort         = 9000
)

// FunctionsBuilder builds the Edge Runtime deployment and service.
type FunctionsBuilder struct {
	ctx *components.TenantContext
}

func NewFunctionsBuilder(ctx *components.TenantContext) *FunctionsBuilder {
	return &FunctionsBuilder{ctx: ctx}
}

func (b *FunctionsBuilder) ComponentName() string {
	return componentFunctions
}

func (b *FunctionsBuilder) resourceName() string {
	return fmt.Sprintf("%s-functions", b.ctx.TenantID())
}

func (b *FunctionsBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentFunctions)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentFunctions)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	preset := resources.GetPreset(b.ctx.Tenant.Spec.Resources)
	functionsSpec := b.ctx.Tenant.Spec.Functions

	env := []corev1.EnvVar{
		{Name: "JWT_SECRET", Value: b.ctx.JWTSecret},
		{Name: "SUPABASE_URL", Value: fmt.Sprintf("https://%s.%s", b.ctx.TenantID(), b.ctx.BaseDomain())},
		{Name: "SUPABASE_ANON_KEY", Value: b.ctx.AnonKey},
		{Name: "SUPABASE_SERVICE_ROLE_KEY", Value: b.ctx.ServiceRoleKey},
		{Name: "SUPABASE_DB_URL", Value: b.ctx.DatabaseDSN("postgres", b.ctx.DatabasePassword)},
		{Name: "VERIFY_JWT", Value: strconv.FormatBool(functionsSpec.VerifyJWT)},
	}

	return resources.NewDeploymentBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithReplicas(preset.Replicas).
		WithPodAnnotations(map[string]string{
			credentials.SecretHashAnnotation: b.ctx.SecretHash,
		}).
		WithContainer(corev1.Container{
			Name:    componentFunctions,
			Image:   defaultFunctionsImage,
			Command: []string{"start", "--main-service", "/home/deno/functions/main"},
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: functionsPort, Protocol: corev1.ProtocolTCP},
			},
			Env:       env,
			Resources: preset.Resources,
		}).
		Build()
}

func (b *FunctionsBuilder) BuildService() *corev1.Service {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentFunctions)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentFunctions)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	return resources.NewServiceBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("http", functionsPort, functionsPort).
		Build()
}

func NewFunctions(ctx *components.TenantContext) *TenantServiceComponent {
	return NewTenantServiceComponent(ctx, NewFunctionsBuilder(ctx))
}
