package usage

type Aggregator interface {
	Merge(...Report) (*Report, error)
}
