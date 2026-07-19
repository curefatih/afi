package controlplane

import (
	"crypto/subtle"
	"errors"
)

var ErrInternalUnauthorized = errors.New("invalid internal token")

// CheckInternalToken validates X-AFI-Internal-Token for HTTP admin endpoints.
// An empty configured token rejects all callers (fail closed).
func CheckInternalToken(configured, presented string) error {
	if configured == "" {
		return ErrInternalUnauthorized
	}
	if subtle.ConstantTimeCompare([]byte(configured), []byte(presented)) != 1 {
		return ErrInternalUnauthorized
	}
	return nil
}
