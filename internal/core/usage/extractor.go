package usage

type Extractor interface {
	Extract(any) (*Report, error)
}
