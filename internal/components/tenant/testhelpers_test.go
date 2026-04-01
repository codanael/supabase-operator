package tenant

import (
	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/codanael/supabase-operator/internal/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestTenant() *v1alpha1.SupabaseTenant {
	return &v1alpha1.SupabaseTenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acme",
			Namespace: "supabase-system",
		},
		Spec: v1alpha1.SupabaseTenantSpec{
			TenantID:    "acme",
			SupabaseRef: "main",
			Auth: v1alpha1.TenantAuthSpec{
				SiteURL: "https://app.acme.com",
			},
		},
	}
}

func newTestSupabase() *v1alpha1.Supabase {
	return &v1alpha1.Supabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "supabase-system",
		},
		Spec: v1alpha1.SupabaseSpec{
			Database: v1alpha1.DatabaseSpec{
				Instances: 3,
				ImageName: "supabase/postgres:15.8.1.085",
				Storage:   v1alpha1.PersistentStorageSpec{Size: "50Gi"},
			},
			Gateway: v1alpha1.GatewaySpec{
				GatewayClassName: "istio",
				BaseDomain:       "supabase.example.com",
			},
		},
	}
}

func newTestTenantContext() *components.TenantContext {
	tenant := newTestTenant()
	supabase := newTestSupabase()
	return &components.TenantContext{
		Tenant:           tenant,
		Supabase:         supabase,
		TenantNamespace:  "supabase-acme",
		DatabaseName:     "acme",
		DatabaseHost:     "main-db-rw.supabase-system.svc.cluster.local",
		DatabasePort:     "5432",
		JWTSecret:        "test-jwt-secret-base64encoded",
		AnonKey:          "test-anon-key",
		ServiceRoleKey:   "test-service-role-key",
		DatabasePassword: "test-db-password",
	}
}

func envToMap(envs []corev1.EnvVar) map[string]string {
	m := make(map[string]string, len(envs))
	for _, e := range envs {
		m[e.Name] = e.Value
	}
	return m
}
