package controlplane

import "github.com/curefatih/afi/internal/access"

// KeyPrefix returns a short display prefix for an API key.
func KeyPrefix(raw string) string {
	return access.Prefix(raw)
}

func HashAPIKey(raw string) string {
	return access.Hash(raw)
}
