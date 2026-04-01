package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestRouting_Name(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewRoutingComponent(tctx)
	assert.Equal(t, "routing", c.Name())
}

func TestRouting_BuildHTTPRoute(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewRoutingComponent(tctx)

	route := c.buildHTTPRoute()

	// Metadata
	assert.Equal(t, "acme-routes", route.Name)
	assert.Equal(t, "supabase-system", route.Namespace)
	assert.Equal(t, "acme", route.Labels["supabase.codanael.io/tenant"])

	// Hostname
	require.Len(t, route.Spec.Hostnames, 1)
	assert.Equal(t, gatewayv1.Hostname("acme.supabase.example.com"), route.Spec.Hostnames[0])

	// ParentRef
	require.Len(t, route.Spec.ParentRefs, 1)
	assert.Equal(t, gatewayv1.ObjectName("main-gateway"), route.Spec.ParentRefs[0].Name)
	require.NotNil(t, route.Spec.ParentRefs[0].Namespace)
	assert.Equal(t, gatewayv1.Namespace("supabase-system"), *route.Spec.ParentRefs[0].Namespace)

	// Must have exactly 5 rules
	require.Len(t, route.Spec.Rules, 5)

	// Verify each rule
	type expectedRule struct {
		path    string
		svcName string
		port    int32
	}
	expected := []expectedRule{
		{"/auth/v1", "acme-auth", 9999},
		{"/rest/v1", "acme-rest", 3000},
		{"/realtime/v1", "acme-realtime", 4000},
		{"/storage/v1", "acme-storage", 5000},
		{"/functions/v1", "acme-functions", 9000},
	}

	for i, exp := range expected {
		rule := route.Spec.Rules[i]

		// Path match
		require.Len(t, rule.Matches, 1, "rule %d should have 1 match", i)
		require.NotNil(t, rule.Matches[0].Path)
		require.NotNil(t, rule.Matches[0].Path.Type)
		assert.Equal(t, gatewayv1.PathMatchPathPrefix, *rule.Matches[0].Path.Type, "rule %d path type", i)
		require.NotNil(t, rule.Matches[0].Path.Value)
		assert.Equal(t, exp.path, *rule.Matches[0].Path.Value, "rule %d path value", i)

		// Backend ref
		require.Len(t, rule.BackendRefs, 1, "rule %d should have 1 backend ref", i)
		backend := rule.BackendRefs[0].BackendObjectReference
		assert.Equal(t, gatewayv1.ObjectName(exp.svcName), backend.Name, "rule %d svc name", i)
		require.NotNil(t, backend.Namespace, "rule %d namespace should be set", i)
		assert.Equal(t, gatewayv1.Namespace("supabase-acme"), *backend.Namespace, "rule %d namespace", i)
		require.NotNil(t, backend.Port, "rule %d port should be set", i)
		assert.Equal(t, exp.port, *backend.Port, "rule %d port", i)
	}
}
