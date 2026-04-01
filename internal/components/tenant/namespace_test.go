package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaceComponent_Name(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewNamespaceComponent(tctx)
	assert.Equal(t, "namespace", c.Name())
}

func TestNamespaceComponent_BuildNamespace(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewNamespaceComponent(tctx)

	ns := c.buildNamespace()

	assert.Equal(t, "supabase-acme", ns.Name)
	assert.Equal(t, "acme", ns.Labels["supabase.codanael.io/tenant"])
	assert.Equal(t, "supabase-operator", ns.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "main", ns.Labels["app.kubernetes.io/instance"])
}

func TestNamespaceComponent_BuildNetworkPolicy(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewNamespaceComponent(tctx)

	np := c.buildNetworkPolicy()

	assert.Equal(t, "deny-cross-tenant", np.Name)
	assert.Equal(t, "supabase-acme", np.Namespace)

	// Should have ingress rules
	require.Len(t, np.Spec.Ingress, 1)
	require.Len(t, np.Spec.Ingress[0].From, 2)

	// First peer: same namespace (PodSelector with empty selector)
	assert.NotNil(t, np.Spec.Ingress[0].From[0].PodSelector)
	assert.Nil(t, np.Spec.Ingress[0].From[0].NamespaceSelector)

	// Second peer: platform namespace
	assert.NotNil(t, np.Spec.Ingress[0].From[1].NamespaceSelector)
	assert.Equal(t, "supabase-system",
		np.Spec.Ingress[0].From[1].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])
}
