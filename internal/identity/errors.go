package identity

import "errors"

// Domain sentinel errors for AuthN.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSSODisabled        = errors.New("sso disabled")
	ErrUnknownProvider    = errors.New("unknown sso provider")
	ErrInvalidSSOState    = errors.New("invalid or expired sso state")
)
