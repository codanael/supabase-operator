package credentials

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateHMACSecret(t *testing.T) {
	secret, err := GenerateHMACSecret(32)
	require.NoError(t, err)
	// base64 of 32 bytes = 44 characters
	assert.Len(t, secret, 44)
}

func TestGenerateAnonKey(t *testing.T) {
	secret, err := GenerateHMACSecret(32)
	require.NoError(t, err)

	key, err := GenerateAnonKey(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, key)
	assert.True(t, strings.Contains(key, "."), "JWT should contain dots")
	// JWT has 3 parts
	assert.Equal(t, 3, len(strings.Split(key, ".")))
}

func TestGenerateServiceRoleKey(t *testing.T) {
	secret, err := GenerateHMACSecret(32)
	require.NoError(t, err)

	key, err := GenerateServiceRoleKey(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, key)
	assert.True(t, strings.Contains(key, "."), "JWT should contain dots")
	assert.Equal(t, 3, len(strings.Split(key, ".")))
}
