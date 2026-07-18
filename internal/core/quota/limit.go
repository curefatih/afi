package quota

import "github.com/curefatih/afi/internal/core/usage"

type Limit struct {
	ID string

	Metric usage.Metric

	Max int64

	Window Window

	Scope Scope

	Enabled bool

	Subject Subject
}
