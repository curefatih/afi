package openai

import "errors"

var (
	ErrUnexpectedStatus = errors.New("unexpected response status")

	ErrInvalidResponse = errors.New("invalid openai response")
)