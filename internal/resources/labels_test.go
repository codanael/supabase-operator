package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlatformLabels(t *testing.T) {
	labels := PlatformLabels("main", "imgproxy")
	assert.Equal(t, "supabase-operator", labels[LabelManagedBy])
	assert.Equal(t, "supabase", labels[LabelPartOf])
	assert.Equal(t, "main", labels[LabelInstance])
	assert.Equal(t, "imgproxy", labels[LabelComponent])
}

func TestTenantLabels(t *testing.T) {
	labels := TenantLabels("main", "acme", "auth")
	assert.Equal(t, "supabase-operator", labels[LabelManagedBy])
	assert.Equal(t, "supabase", labels[LabelPartOf])
	assert.Equal(t, "main", labels[LabelInstance])
	assert.Equal(t, "auth", labels[LabelComponent])
	assert.Equal(t, "acme", labels[LabelTenant])
}

func TestSelectorLabels(t *testing.T) {
	labels := SelectorLabels("main", "rest")
	assert.Equal(t, "main", labels[LabelInstance])
	assert.Equal(t, "rest", labels[LabelComponent])
	_, hasManagedBy := labels[LabelManagedBy]
	assert.False(t, hasManagedBy, "selector labels should not include managed-by")
}
