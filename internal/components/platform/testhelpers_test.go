package platform

import (
	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/codanael/supabase-operator/internal/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func newTestPlatformContext(sb *v1alpha1.Supabase) *components.PlatformContext {
	return &components.PlatformContext{Supabase: sb}
}
