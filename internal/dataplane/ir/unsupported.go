package ir

import (
	"errors"
	"fmt"
)

// Gateway dialect IR capabilities for the current chat IR version.
// Catalog advertising and request validation must stay in sync with these.
const (
	DialectSupportsTools  = true
	DialectSupportsVision = true
)

// UnsupportedError is returned when a client request uses a feature chat IR cannot serve yet.
type UnsupportedError struct {
	Feature string
	Message string
}

func (e *UnsupportedError) Error() string {
	if e == nil {
		return "unsupported feature"
	}
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("unsupported feature: %s", e.Feature)
}

// Unsupported returns a new UnsupportedError.
func Unsupported(feature, message string) error {
	return &UnsupportedError{Feature: feature, Message: message}
}

// AsUnsupported extracts UnsupportedError from err.
func AsUnsupported(err error) (*UnsupportedError, bool) {
	var u *UnsupportedError
	if errors.As(err, &u) {
		return u, true
	}
	return nil, false
}
