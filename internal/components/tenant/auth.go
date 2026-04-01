package tenant

import (
	"fmt"
	"strconv"

	"github.com/codanael/supabase-operator/internal/components"
	"github.com/codanael/supabase-operator/internal/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	componentAuth    = "auth"
	defaultAuthImage = "supabase/gotrue:v2.186.0"
	authPort         = 9999
)

// AuthBuilder builds the GoTrue auth deployment and service.
type AuthBuilder struct {
	ctx *components.TenantContext
}

func NewAuthBuilder(ctx *components.TenantContext) *AuthBuilder {
	return &AuthBuilder{ctx: ctx}
}

func (b *AuthBuilder) ComponentName() string {
	return componentAuth
}

func (b *AuthBuilder) resourceName() string {
	return fmt.Sprintf("%s-auth", b.ctx.TenantID())
}

func (b *AuthBuilder) BuildDeployment() *appsv1.Deployment {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentAuth)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentAuth)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	// Read DB credentials secret key for auth-admin-password
	dbPassword := b.ctx.DatabasePassword
	authSpec := b.ctx.Tenant.Spec.Auth

	env := []corev1.EnvVar{
		{Name: "GOTRUE_API_HOST", Value: "0.0.0.0"},
		{Name: "GOTRUE_API_PORT", Value: strconv.Itoa(authPort)},
		{Name: "GOTRUE_DB_DRIVER", Value: "postgres"},
		{Name: "GOTRUE_DB_DATABASE_URL", Value: b.ctx.DatabaseDSN("supabase_auth_admin", dbPassword)},
		{Name: "GOTRUE_SITE_URL", Value: authSpec.SiteURL},
		{Name: "GOTRUE_JWT_SECRET", Value: b.ctx.JWTSecret},
		{Name: "GOTRUE_JWT_AUD", Value: "authenticated"},
		{Name: "GOTRUE_JWT_DEFAULT_GROUP_NAME", Value: "authenticated"},
		{Name: "GOTRUE_JWT_ADMIN_ROLES", Value: "service_role"},
		{Name: "GOTRUE_JWT_EXP", Value: "3600"},
		{Name: "GOTRUE_DISABLE_SIGNUP", Value: strconv.FormatBool(authSpec.DisableSignup)},
		{Name: "API_EXTERNAL_URL", Value: fmt.Sprintf("https://%s.%s", b.ctx.TenantID(), b.ctx.BaseDomain())},
	}

	if authSpec.Email != nil {
		env = append(env,
			corev1.EnvVar{Name: "GOTRUE_EXTERNAL_EMAIL_ENABLED", Value: strconv.FormatBool(authSpec.Email.Enabled)},
			corev1.EnvVar{Name: "GOTRUE_MAILER_AUTOCONFIRM", Value: strconv.FormatBool(authSpec.Email.Autoconfirm)},
		)
	}

	return resources.NewDeploymentBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithContainer(corev1.Container{
			Name:  componentAuth,
			Image: defaultAuthImage,
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: authPort, Protocol: corev1.ProtocolTCP},
			},
			Env: env,
		}).
		Build()
}

func (b *AuthBuilder) BuildService() *corev1.Service {
	labels := resources.TenantLabels(b.ctx.InstanceName(), b.ctx.TenantID(), componentAuth)
	selectorLabels := resources.SelectorLabels(b.ctx.InstanceName(), componentAuth)
	selectorLabels[resources.LabelTenant] = b.ctx.TenantID()

	return resources.NewServiceBuilder(b.ctx.TenantNamespace, b.resourceName()).
		WithLabels(labels).
		WithSelectorLabels(selectorLabels).
		WithPort("http", authPort, authPort).
		Build()
}

func NewAuth(ctx *components.TenantContext) *TenantServiceComponent {
	return NewTenantServiceComponent(ctx, NewAuthBuilder(ctx))
}
