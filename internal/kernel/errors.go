package kernel

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInvalidRequest = errors.New("invalid request")
	ErrConflict       = errors.New("conflict")
)
