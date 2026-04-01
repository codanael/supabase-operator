package resources

import (
	"testing"

	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestGetPreset_Small(t *testing.T) {
	small := GetPreset(v1alpha1.ResourcePresetSmall)
	assert.Equal(t, int32(1), small.Replicas)
	assert.Equal(t, "100m", small.Resources.Requests.Cpu().String())
	assert.Equal(t, "128Mi", small.Resources.Requests.Memory().String())
	assert.Equal(t, "500m", small.Resources.Limits.Cpu().String())
	assert.Equal(t, "512Mi", small.Resources.Limits.Memory().String())
}

func TestGetPreset_Medium(t *testing.T) {
	medium := GetPreset(v1alpha1.ResourcePresetMedium)
	assert.Equal(t, int32(2), medium.Replicas)
	assert.Equal(t, "250m", medium.Resources.Requests.Cpu().String())
	assert.Equal(t, "256Mi", medium.Resources.Requests.Memory().String())
	assert.Equal(t, "1", medium.Resources.Limits.Cpu().String())
	assert.Equal(t, "1Gi", medium.Resources.Limits.Memory().String())
}

func TestGetPreset_Large(t *testing.T) {
	large := GetPreset(v1alpha1.ResourcePresetLarge)
	assert.Equal(t, int32(3), large.Replicas)
	assert.Equal(t, "500m", large.Resources.Requests.Cpu().String())
	assert.Equal(t, "512Mi", large.Resources.Requests.Memory().String())
	assert.Equal(t, "2", large.Resources.Limits.Cpu().String())
	assert.Equal(t, "2Gi", large.Resources.Limits.Memory().String())
}

func TestGetPreset_Custom(t *testing.T) {
	custom := GetPreset(v1alpha1.ResourcePresetCustom)
	assert.Equal(t, int32(1), custom.Replicas)
	assert.True(t, custom.Resources.Requests == nil || len(custom.Resources.Requests) == 0)
	assert.True(t, custom.Resources.Limits == nil || len(custom.Resources.Limits) == 0)
}

func TestGetPreset_Unknown(t *testing.T) {
	unknown := GetPreset(v1alpha1.ResourcePreset("unknown"))
	assert.Equal(t, int32(1), unknown.Replicas)
}
