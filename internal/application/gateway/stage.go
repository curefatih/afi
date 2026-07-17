package gateway

type Stage interface {
	Execute(*Context) error
}