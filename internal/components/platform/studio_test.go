package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStudio_Name(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewStudio(pctx)
	assert.Equal(t, "studio", c.Name())
}

func TestStudio_BuildDeployment(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewStudioBuilder(pctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "main-studio", deploy.Name)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "supabase/studio:2026.03.16-sha-5528817", c.Image)
	assert.Equal(t, int32(3000), c.Ports[0].ContainerPort)

	envMap := envToMap(c.Env)
	assert.Equal(t, "::", envMap["HOSTNAME"])
	assert.Equal(t, "3000", envMap["STUDIO_PORT"])
	assert.Equal(t, "true", envMap["NEXT_PUBLIC_ENABLE_LOGS"])
	assert.Equal(t, "postgres", envMap["NEXT_ANALYTICS_BACKEND_PROVIDER"])
}

func TestStudio_BuildService(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewStudioBuilder(pctx)

	svc := b.BuildService()

	assert.Equal(t, "main-studio", svc.Name)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(3000), svc.Spec.Ports[0].Port)
}
