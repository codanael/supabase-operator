package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupavisor_Name(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewSupavisor(pctx)
	assert.Equal(t, "supavisor", c.Name())
}

func TestSupavisor_BuildDeployment(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewSupavisorBuilder(pctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "main-supavisor", deploy.Name)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "supabase/supavisor:2.7.4", c.Image)
	require.Len(t, c.Ports, 3)
	assert.Equal(t, int32(4000), c.Ports[0].ContainerPort)
	assert.Equal(t, int32(5432), c.Ports[1].ContainerPort)
	assert.Equal(t, int32(6543), c.Ports[2].ContainerPort)

	envMap := envToMap(c.Env)
	assert.Equal(t, "4000", envMap["PORT"])
	assert.Equal(t, "true", envMap["CLUSTER_POSTGRES"])
	assert.Equal(t, "local", envMap["REGION"])
	assert.Equal(t, "-proto_dist inet_tcp", envMap["ERL_AFLAGS"])
	assert.Equal(t, "transaction", envMap["POOLER_POOL_MODE"])
}

func TestSupavisor_BuildService(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewSupavisorBuilder(pctx)

	svc := b.BuildService()

	assert.Equal(t, "main-supavisor", svc.Name)
	require.Len(t, svc.Spec.Ports, 3)
	assert.Equal(t, int32(4000), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(5432), svc.Spec.Ports[1].Port)
	assert.Equal(t, int32(6543), svc.Spec.Ports[2].Port)
}
