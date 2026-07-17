package quota

type Decision struct {
	Allowed bool

	Reason string

	Violated *Quota
}
