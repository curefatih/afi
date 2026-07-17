package provider

type Registry interface {
	Register(provider Provider, client Client)

	Client(providerID string) (Client, error)

	List() []Provider
}
