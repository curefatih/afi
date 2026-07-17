package plugin

type Executor interface {
	Run(
		ctx *Context,
		hook Hook,
	) (*Result, error)
}
