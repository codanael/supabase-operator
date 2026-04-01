package credentials

import (
	"crypto/sha256"
	"fmt"
	"sort"
)

const SecretHashAnnotation = "supabase.codanael.io/secret-hash"

// ComputeSecretHash computes a SHA-256 hash from the combined data of multiple
// secret data maps. This is used to detect when secrets change so that dependent
// Deployments can be rolling-restarted via a pod template annotation change.
func ComputeSecretHash(secretDataMaps ...map[string][]byte) string {
	h := sha256.New()
	for _, data := range secretDataMaps {
		// Sort keys for deterministic output
		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.Write([]byte(k))
			h.Write(data[k])
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
