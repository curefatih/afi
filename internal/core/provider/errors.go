package provider

import "errors"

var (
	ErrProviderNotFound = errors.New("provider not found")

	ErrModelNotFound = errors.New("model not found")

	ErrCapabilityUnsupported = errors.New("capability not supported")
)
