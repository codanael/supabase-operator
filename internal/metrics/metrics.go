package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	TenantsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "supabase_tenants_total",
		Help: "Total number of SupabaseTenant resources",
	})

	TenantsReady = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "supabase_tenants_ready",
		Help: "Number of SupabaseTenant resources in Ready phase",
	})

	TenantsSuspended = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "supabase_tenants_suspended",
		Help: "Number of suspended tenants",
	})

	ReconcileDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "supabase_reconcile_duration_seconds",
		Help:    "Duration of reconciliation loops",
		Buckets: prometheus.DefBuckets,
	}, []string{"controller", "result"})
)

func init() {
	metrics.Registry.MustRegister(
		TenantsTotal,
		TenantsReady,
		TenantsSuspended,
		ReconcileDuration,
	)
}
