package database

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderInitScripts(t *testing.T) {
	params := InitParams{
		DatabaseName:          "test-tenant",
		JWTSecret:             "super-secret-jwt-key",
		AuthenticatorPassword: "auth-pass-123",
		AuthAdminPassword:     "auth-admin-pass-456",
		StorageAdminPassword:  "storage-admin-pass-789",
	}

	scripts, err := RenderInitScripts(params)
	require.NoError(t, err)
	require.NotEmpty(t, scripts)

	combined := strings.Join(scripts, "\n")

	// Verify template variables were rendered
	assert.Contains(t, combined, "test-tenant")
	assert.Contains(t, combined, "super-secret-jwt-key")
	assert.Contains(t, combined, "auth-pass-123")
	assert.Contains(t, combined, "auth-admin-pass-456")
	assert.Contains(t, combined, "storage-admin-pass-789")

	// Verify no unrendered template placeholders remain
	assert.NotContains(t, combined, "{{.")

	// Verify known SQL content is present
	assert.Contains(t, combined, "supabase_functions")
	assert.Contains(t, combined, "_realtime")
	assert.Contains(t, combined, "app.settings.jwt_secret")
	assert.Contains(t, combined, "authenticator")
}

func TestCombinedInitSQL(t *testing.T) {
	params := InitParams{
		DatabaseName:          "mydb",
		JWTSecret:             "jwtsecret",
		AuthenticatorPassword: "authpw",
		AuthAdminPassword:     "authadminpw",
		StorageAdminPassword:  "storagepw",
	}

	sql, err := CombinedInitSQL(params)
	require.NoError(t, err)
	assert.NotEmpty(t, sql)
	assert.NotContains(t, sql, "{{.")
}
