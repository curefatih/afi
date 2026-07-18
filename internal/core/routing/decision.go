package routing

type Decision struct {
	Route  Route
	Target Target

	Metadata map[string]string
}
