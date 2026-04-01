package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalytics_Name(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewAnalytics(pctx)
	assert.Equal(t, "analytics", c.Name())
}

func TestAnalytics_BuildDeployment(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewAnalyticsBuilder(pctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "main-analytics", deploy.Name)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "supabase/logflare:1.31.2", c.Image)
	assert.Equal(t, int32(4000), c.Ports[0].ContainerPort)

	envMap := envToMap(c.Env)
	assert.Equal(t, "127.0.0.1", envMap["LOGFLARE_NODE_HOST"])
	assert.Equal(t, "true", envMap["LOGFLARE_SINGLE_TENANT"])
	assert.Equal(t, "true", envMap["LOGFLARE_SUPABASE_MODE"])
	assert.Equal(t, "multibackend=true", envMap["LOGFLARE_FEATURE_FLAG_OVERRIDE"])
	assert.Equal(t, "_analytics", envMap["DB_SCHEMA"])
	assert.Equal(t, "_analytics", envMap["POSTGRES_BACKEND_SCHEMA"])
}

func TestAnalytics_BuildService(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewAnalyticsBuilder(pctx)

	svc := b.BuildService()

	assert.Equal(t, "main-analytics", svc.Name)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(4000), svc.Spec.Ports[0].Port)
}
