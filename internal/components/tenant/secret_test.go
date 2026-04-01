package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretComponent_Name(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewSecretComponent(tctx)
	assert.Equal(t, "secrets", c.Name())
}

func TestSecretComponent_BuildJWTSecret(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewSecretComponent(tctx)

	secret, err := c.buildJWTSecret()
	require.NoError(t, err)

	assert.Equal(t, "acme-jwt", secret.Name)
	assert.Equal(t, "supabase-acme", secret.Namespace)

	// Must have all three keys
	assert.Contains(t, secret.Data, "jwt-secret")
	assert.Contains(t, secret.Data, "anon-key")
	assert.Contains(t, secret.Data, "service-role-key")

	// Values must be non-empty
	assert.NotEmpty(t, secret.Data["jwt-secret"])
	assert.NotEmpty(t, secret.Data["anon-key"])
	assert.NotEmpty(t, secret.Data["service-role-key"])

	// Labels
	assert.Equal(t, "acme", secret.Labels["supabase.codanael.io/tenant"])
	assert.Equal(t, "supabase-operator", secret.Labels["app.kubernetes.io/managed-by"])
}

func TestSecretComponent_BuildDBCredentialsSecret(t *testing.T) {
	tctx := newTestTenantContext()
	c := NewSecretComponent(tctx)

	secret, err := c.buildDBCredentialsSecret()
	require.NoError(t, err)

	assert.Equal(t, "acme-db-credentials", secret.Name)
	assert.Equal(t, "supabase-acme", secret.Namespace)

	// Must have all four keys
	assert.Contains(t, secret.Data, "postgres-password")
	assert.Contains(t, secret.Data, "authenticator-password")
	assert.Contains(t, secret.Data, "auth-admin-password")
	assert.Contains(t, secret.Data, "storage-admin-password")

	// Values must be non-empty
	assert.NotEmpty(t, secret.Data["postgres-password"])
	assert.NotEmpty(t, secret.Data["authenticator-password"])
	assert.NotEmpty(t, secret.Data["auth-admin-password"])
	assert.NotEmpty(t, secret.Data["storage-admin-password"])

	// All passwords should be different
	passwords := make(map[string]bool)
	for _, v := range secret.Data {
		passwords[string(v)] = true
	}
	assert.Equal(t, 4, len(passwords), "all passwords should be unique")
}
