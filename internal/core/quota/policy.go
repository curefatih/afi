package quota

import "github.com/curefatih/afi/internal/core/usage"

type Decision struct {
	Allowed bool

	Metric usage.Metric

	Quota *Quota
}

type Subject struct {
	Type string

	ID string
}
