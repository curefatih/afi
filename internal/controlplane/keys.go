package controlplane

import "github.com/curefatih/afi/internal/snapshot"

// KeyPrefix returns a short display prefix for an API key.
func KeyPrefix(raw string) string {
	const n = 10
	if len(raw) <= n {
		return raw
	}
	return raw[:n]
}

func HashAPIKey(raw string) string {
	return snapshot.HashKey(raw)
}
