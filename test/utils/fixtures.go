package utils

import (
	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewTestSupabase(name, namespace string) *v1alpha1.Supabase {
	return &v1alpha1.Supabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.SupabaseSpec{
			Database: v1alpha1.DatabaseSpec{
				Instances: 1,
				ImageName: "supabase/postgres:15.8.1.085",
				Storage:   v1alpha1.PersistentStorageSpec{Size: "1Gi"},
			},
			Gateway: v1alpha1.GatewaySpec{
				GatewayClassName: "test-gateway-class",
				BaseDomain:       "supabase.test",
			},
		},
	}
}
