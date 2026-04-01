package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_Name(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewAuth(tctx)
	assert.Equal(t, "auth", c.Name())
}

func TestAuth_BuildDeployment(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewAuthBuilder(tctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "acme-auth", deploy.Name)
	assert.Equal(t, "supabase-acme", deploy.Namespace)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "supabase/gotrue:v2.186.0", c.Image)
	assert.Equal(t, int32(9999), c.Ports[0].ContainerPort)

	envMap := envToMap(c.Env)
	assert.Equal(t, "0.0.0.0", envMap["GOTRUE_API_HOST"])
	assert.Equal(t, "9999", envMap["GOTRUE_API_PORT"])
	assert.Equal(t, "postgres", envMap["GOTRUE_DB_DRIVER"])
	assert.Contains(t, envMap["GOTRUE_DB_DATABASE_URL"], "supabase_auth_admin")
	assert.Equal(t, "https://app.acme.com", envMap["GOTRUE_SITE_URL"])
	assert.Equal(t, "test-jwt-secret-base64encoded", envMap["GOTRUE_JWT_SECRET"])
	assert.Equal(t, "authenticated", envMap["GOTRUE_JWT_AUD"])
	assert.Equal(t, "service_role", envMap["GOTRUE_JWT_ADMIN_ROLES"])
	assert.Equal(t, "3600", envMap["GOTRUE_JWT_EXP"])
	assert.Equal(t, "https://acme.supabase.example.com", envMap["API_EXTERNAL_URL"])
}

func TestAuth_BuildService(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewAuthBuilder(tctx)

	svc := b.BuildService()

	assert.Equal(t, "acme-auth", svc.Name)
	assert.Equal(t, "supabase-acme", svc.Namespace)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(9999), svc.Spec.Ports[0].Port)
}
