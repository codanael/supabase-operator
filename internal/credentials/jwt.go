package credentials

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateHMACSecret generates a random base64-encoded secret of the given byte length.
func GenerateHMACSecret(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating HMAC secret: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// GenerateAnonKey generates a JWT with role=anon signed with the given HMAC secret.
func GenerateAnonKey(jwtSecret string) (string, error) {
	return generateRoleKey(jwtSecret, "anon")
}

// GenerateServiceRoleKey generates a JWT with role=service_role signed with the given HMAC secret.
func GenerateServiceRoleKey(jwtSecret string) (string, error) {
	return generateRoleKey(jwtSecret, "service_role")
}

func generateRoleKey(jwtSecret, role string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"role": role,
		"iss":  "supabase-operator",
		"iat":  now.Unix(),
		"exp":  now.Add(10 * 365 * 24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("signing %s key: %w", role, err)
	}
	return signed, nil
}
