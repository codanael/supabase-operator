package credentials

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
)

// GeneratePassword generates a random base64-encoded password of the given byte length.
func GeneratePassword(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating password: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// SecretHash computes a deterministic truncated SHA256 hash (16 hex chars) over
// the sorted key-value pairs in a Secret's data map. This is useful for detecting
// when a Secret's content has changed.
func SecretHash(data map[string][]byte) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte("="))
		h.Write(data[k])
		h.Write([]byte("\n"))
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
