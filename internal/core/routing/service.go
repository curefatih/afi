package routing

type Service struct {
	repo Repository

	selector Selector
}
