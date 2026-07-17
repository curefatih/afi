package routing

type Policy interface {
	Select(
		Route,
	) (*Target, error)
}