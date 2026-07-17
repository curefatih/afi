package provider

type Translator interface {
	Encode(*Request) (any, error)

	Decode(any) (*Response, error)
}
