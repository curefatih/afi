package plugin

type Registry interface {
	Plugins(hook Hook) []Executor
}
