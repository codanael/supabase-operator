package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctions_Name(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewFunctions(tctx)
	assert.Equal(t, "functions", c.Name())
}

func TestFunctions_BuildDeployment(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewFunctionsBuilder(tctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "acme-functions", deploy.Name)
	assert.Equal(t, "supabase-acme", deploy.Namespace)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "supabase/edge-runtime:v1.71.2", c.Image)
	assert.Equal(t, int32(9000), c.Ports[0].ContainerPort)

	// Verify command
	assert.Equal(t, []string{"start", "--main-service", "/home/deno/functions/main"}, c.Command)

	envMap := envToMap(c.Env)
	assert.Equal(t, "test-jwt-secret-base64encoded", envMap["JWT_SECRET"])
	assert.Equal(t, "https://acme.supabase.example.com", envMap["SUPABASE_URL"])
	assert.Equal(t, "test-anon-key", envMap["SUPABASE_ANON_KEY"])
	assert.Equal(t, "test-service-role-key", envMap["SUPABASE_SERVICE_ROLE_KEY"])
	assert.Contains(t, envMap["SUPABASE_DB_URL"], "postgres")
	assert.Equal(t, "false", envMap["VERIFY_JWT"]) // default is false for zero-value bool
}

func TestFunctions_BuildService(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewFunctionsBuilder(tctx)

	svc := b.BuildService()

	assert.Equal(t, "acme-functions", svc.Name)
	assert.Equal(t, "supabase-acme", svc.Namespace)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(9000), svc.Spec.Ports[0].Port)
}
