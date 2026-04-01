package platform

import (
	"testing"

	v1alpha1 "github.com/codanael/supabase-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestGateway_Name(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	g := NewGateway(pctx)
	assert.Equal(t, "gateway", g.Name())
}

func TestGateway_BuildGateway(t *testing.T) {
	sb := newTestSupabase()
	pctx := newTestPlatformContext(sb)
	g := NewGateway(pctx)

	gw := g.buildGateway()

	assert.Equal(t, "main-gateway", gw.Name)
	assert.Equal(t, "supabase-system", gw.Namespace)
	assert.Equal(t, gatewayv1.ObjectName("istio"), gw.Spec.GatewayClassName)

	require.Len(t, gw.Spec.Listeners, 1)
	listener := gw.Spec.Listeners[0]
	assert.Equal(t, gatewayv1.SectionName("https"), listener.Name)
	assert.Equal(t, gatewayv1.PortNumber(443), listener.Port)
	assert.Equal(t, gatewayv1.HTTPSProtocolType, listener.Protocol)
	require.NotNil(t, listener.Hostname)
	assert.Equal(t, gatewayv1.Hostname("*.supabase.example.com"), *listener.Hostname)

	// No TLS config when spec.gateway.tls is nil
	assert.Nil(t, listener.TLS)

	// Check labels
	assert.Equal(t, "supabase-operator", gw.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "main", gw.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, "gateway", gw.Labels["app.kubernetes.io/component"])
}

func TestGateway_BuildGateway_WithTLS(t *testing.T) {
	sb := newTestSupabase()
	sb.Spec.Gateway.TLS = &v1alpha1.GatewayTLSSpec{
		CertificateSecretRef: "wildcard-cert",
	}
	pctx := newTestPlatformContext(sb)
	g := NewGateway(pctx)

	gw := g.buildGateway()

	require.Len(t, gw.Spec.Listeners, 1)
	listener := gw.Spec.Listeners[0]
	require.NotNil(t, listener.TLS)
	assert.Equal(t, gatewayv1.TLSModeTerminate, *listener.TLS.Mode)
	require.Len(t, listener.TLS.CertificateRefs, 1)
	assert.Equal(t, gatewayv1.ObjectName("wildcard-cert"), listener.TLS.CertificateRefs[0].Name)
}
