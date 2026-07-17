package quota

type Limit struct {
	ID string

	Metric Metric

	Max int64

	Window Window

	Scope Scope

	Enabled bool

	Subject Subject
}
