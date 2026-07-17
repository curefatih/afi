package quota

type Decision struct {
	Allowed bool

	Reason string

	Violated *Quota
}

type Subject struct {
	Type string

	ID string
}
