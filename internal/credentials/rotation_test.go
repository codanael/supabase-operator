package credentials

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeSecretHash_Deterministic(t *testing.T) {
	data := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	hash1 := ComputeSecretHash(data)
	hash2 := ComputeSecretHash(data)

	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64) // SHA-256 hex
}

func TestComputeSecretHash_DifferentData(t *testing.T) {
	data1 := map[string][]byte{
		"key": []byte("value1"),
	}
	data2 := map[string][]byte{
		"key": []byte("value2"),
	}

	hash1 := ComputeSecretHash(data1)
	hash2 := ComputeSecretHash(data2)

	assert.NotEqual(t, hash1, hash2)
}

func TestComputeSecretHash_MultipleMaps(t *testing.T) {
	jwt := map[string][]byte{
		"jwt-secret": []byte("secret123"),
	}
	db := map[string][]byte{
		"password": []byte("pass456"),
	}

	combined := ComputeSecretHash(jwt, db)
	assert.NotEmpty(t, combined)

	// Changing one map changes the hash
	db2 := map[string][]byte{
		"password": []byte("newpass"),
	}
	combined2 := ComputeSecretHash(jwt, db2)
	assert.NotEqual(t, combined, combined2)
}

func TestComputeSecretHash_OrderIndependentKeys(t *testing.T) {
	// Maps with same data but potentially different iteration order
	// should produce the same hash (keys are sorted)
	data := map[string][]byte{
		"z-key": []byte("z-value"),
		"a-key": []byte("a-value"),
		"m-key": []byte("m-value"),
	}

	hash1 := ComputeSecretHash(data)
	hash2 := ComputeSecretHash(data)

	assert.Equal(t, hash1, hash2)
}

func TestSecretHashAnnotation(t *testing.T) {
	assert.Equal(t, "supabase.codanael.io/secret-hash", SecretHashAnnotation)
}
