package identity

import "errors"

// Domain sentinel errors for AuthN.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSSODisabled        = errors.New("sso disabled")
	ErrSignupDisabled     = errors.New("signup disabled")
	ErrUnknownProvider    = errors.New("unknown sso provider")
	ErrInvalidSSOState    = errors.New("invalid or expired sso state")
	ErrInvalidResetToken  = errors.New("invalid or expired reset token")
)
