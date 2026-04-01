package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricsRegistered(t *testing.T) {
	// Verify metrics can be collected without panic
	assert.NotNil(t, TenantsTotal)
	assert.NotNil(t, TenantsReady)
	assert.NotNil(t, TenantsSuspended)
	assert.NotNil(t, ReconcileDuration)
}

func TestGaugeOperations(t *testing.T) {
	// Verify we can set gauge values without panic
	TenantsTotal.Set(5)
	TenantsReady.Set(3)
	TenantsSuspended.Set(1)
}

func TestHistogramObserve(t *testing.T) {
	// Verify we can observe histogram values without panic
	ReconcileDuration.WithLabelValues("supabase", "success").Observe(0.5)
	ReconcileDuration.WithLabelValues("supabasetenant", "success").Observe(1.0)
	ReconcileDuration.WithLabelValues("supabasetenant", "error").Observe(0.1)
}
