package quota

import "errors"

var (
	ErrQuotaExceeded = errors.New("quota exceeded")
)
