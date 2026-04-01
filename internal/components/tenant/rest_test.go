package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestREST_Name(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewREST(tctx)
	assert.Equal(t, "rest", c.Name())
}

func TestREST_BuildDeployment(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewRESTBuilder(tctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "acme-rest", deploy.Name)
	assert.Equal(t, "supabase-acme", deploy.Namespace)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "postgrest/postgrest:v14.6", c.Image)
	assert.Equal(t, int32(3000), c.Ports[0].ContainerPort)

	envMap := envToMap(c.Env)
	assert.Contains(t, envMap["PGRST_DB_URI"], "authenticator")
	assert.Equal(t, "public,graphql_public", envMap["PGRST_DB_SCHEMAS"])
	assert.Equal(t, "1000", envMap["PGRST_DB_MAX_ROWS"])
	assert.Equal(t, "anon", envMap["PGRST_DB_ANON_ROLE"])
	assert.Equal(t, "test-jwt-secret-base64encoded", envMap["PGRST_JWT_SECRET"])
	assert.Equal(t, "false", envMap["PGRST_DB_USE_LEGACY_GUCS"])
	assert.Equal(t, "3600", envMap["PGRST_APP_SETTINGS_JWT_EXP"])
}

func TestREST_BuildService(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewRESTBuilder(tctx)

	svc := b.BuildService()

	assert.Equal(t, "acme-rest", svc.Name)
	assert.Equal(t, "supabase-acme", svc.Namespace)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(3000), svc.Spec.Ports[0].Port)
}
