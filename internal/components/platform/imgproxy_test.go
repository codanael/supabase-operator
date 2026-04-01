package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImgproxy_Name(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewImgproxy(pctx)
	assert.Equal(t, "imgproxy", c.Name())
}

func TestImgproxy_BuildDeployment(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewImgproxyBuilder(pctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "main-imgproxy", deploy.Name)
	assert.Equal(t, "supabase-system", deploy.Namespace)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "darthsim/imgproxy:v3.30.1", c.Image)
	assert.Equal(t, int32(5001), c.Ports[0].ContainerPort)

	// Check env vars
	envMap := envToMap(c.Env)
	assert.Equal(t, ":5001", envMap["IMGPROXY_BIND"])
	assert.Equal(t, "/", envMap["IMGPROXY_LOCAL_FILESYSTEM_ROOT"])
	assert.Equal(t, "true", envMap["IMGPROXY_USE_ETAG"])
	assert.Equal(t, "true", envMap["IMGPROXY_ENABLE_WEBP_DETECTION"])

	// Check labels
	assert.Equal(t, "imgproxy", deploy.Labels["app.kubernetes.io/component"])
}

func TestImgproxy_BuildService(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewImgproxyBuilder(pctx)

	svc := b.BuildService()

	assert.Equal(t, "main-imgproxy", svc.Name)
	assert.Equal(t, "supabase-system", svc.Namespace)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(5001), svc.Spec.Ports[0].Port)
}

func TestImgproxy_ImageOverride(t *testing.T) {
	sb := newTestSupabase()
	customImage := "my-registry/imgproxy:custom"
	sb.Spec.Images.Imgproxy = &customImage
	pctx := newTestPlatformContext(sb)
	b := NewImgproxyBuilder(pctx)

	deploy := b.BuildDeployment()

	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "my-registry/imgproxy:custom", deploy.Spec.Template.Spec.Containers[0].Image)
}
