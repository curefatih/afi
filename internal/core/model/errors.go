package model

import "errors"

var (
	ErrModelNotFound = errors.New("model not found")

	ErrAliasNotFound = errors.New("alias not found")

	ErrCapabilityNotSupported = errors.New("capability not supported")
)