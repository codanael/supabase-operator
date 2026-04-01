package credentials

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePassword(t *testing.T) {
	pw, err := GeneratePassword(24)
	require.NoError(t, err)
	// base64 of 24 bytes = 32 characters
	assert.Len(t, pw, 32)
}

func TestGeneratePasswordUniqueness(t *testing.T) {
	pw1, err := GeneratePassword(24)
	require.NoError(t, err)

	pw2, err := GeneratePassword(24)
	require.NoError(t, err)

	assert.NotEqual(t, pw1, pw2, "two generated passwords should differ")
}

func TestSecretHash(t *testing.T) {
	data := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	hash1 := SecretHash(data)
	hash2 := SecretHash(data)

	assert.Equal(t, hash1, hash2, "hash should be deterministic")
	assert.Len(t, hash1, 16, "hash should be 16 hex characters")

	// Different data should produce different hash
	data2 := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("changed"),
	}
	hash3 := SecretHash(data2)
	assert.NotEqual(t, hash1, hash3, "different data should produce different hash")
}
