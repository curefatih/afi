package auth

import "errors"

var (
	ErrUnauthorized = errors.New("invalid or deactivated api key")
)
