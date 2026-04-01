package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealtime_Name(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewRealtime(tctx)
	assert.Equal(t, "realtime", c.Name())
}

func TestRealtime_BuildDeployment(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewRealtimeBuilder(tctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "acme-realtime", deploy.Name)
	assert.Equal(t, "supabase-acme", deploy.Namespace)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "supabase/realtime:v2.76.5", c.Image)
	assert.Equal(t, int32(4000), c.Ports[0].ContainerPort)

	envMap := envToMap(c.Env)
	assert.Equal(t, "4000", envMap["PORT"])
	assert.Equal(t, "main-db-rw.supabase-system.svc.cluster.local", envMap["DB_HOST"])
	assert.Equal(t, "5432", envMap["DB_PORT"])
	assert.Equal(t, "supabase_admin", envMap["DB_USER"])
	assert.Equal(t, "test-db-password", envMap["DB_PASSWORD"])
	assert.Equal(t, "acme", envMap["DB_NAME"])
	assert.Equal(t, "SET search_path TO _realtime", envMap["DB_AFTER_CONNECT_QUERY"])
	assert.Equal(t, "supabaserealtime", envMap["DB_ENC_KEY"])
	assert.Equal(t, "test-jwt-secret-base64encoded", envMap["API_JWT_SECRET"])
	assert.Equal(t, "realtime", envMap["APP_NAME"])
	assert.Equal(t, "true", envMap["SEED_SELF_HOST"])
}

func TestRealtime_BuildService(t *testing.T) {
	tctx := newTestTenantContext()
	b := NewRealtimeBuilder(tctx)

	svc := b.BuildService()

	assert.Equal(t, "acme-realtime", svc.Name)
	assert.Equal(t, "supabase-acme", svc.Namespace)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(4000), svc.Spec.Ports[0].Port)
}
