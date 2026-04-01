package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVector_Name(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	c := NewVector(pctx)
	assert.Equal(t, "vector", c.Name())
}

func TestVector_BuildDeployment(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewVectorBuilder(pctx)

	deploy := b.BuildDeployment()

	assert.Equal(t, "main-vector", deploy.Name)
	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)

	c := deploy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "timberio/vector:0.53.0-alpine", c.Image)
	assert.Equal(t, []string{"--config", "/etc/vector/vector.yml"}, c.Args)
	assert.Equal(t, int32(9001), c.Ports[0].ContainerPort)
}

func TestVector_BuildService(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	b := NewVectorBuilder(pctx)

	svc := b.BuildService()

	assert.Equal(t, "main-vector", svc.Name)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(9001), svc.Spec.Ports[0].Port)
}
