package gateway

type Executor struct {
	pipeline *Pipeline
}

func NewExecutor(
	pipeline *Pipeline,
) *Executor {

	return &Executor{
		pipeline: pipeline,
	}
}

func (e *Executor) Execute(
	ctx *Context,
) (*Response, error) {

	if err := e.pipeline.Execute(ctx); err != nil {
		return nil, err
	}

	return &Response{
		Response: ctx.ProviderResponse,
		Usage:    ctx.Usage,
		Cost:     ctx.Cost,
	}, nil
}