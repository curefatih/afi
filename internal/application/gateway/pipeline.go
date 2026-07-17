package gateway

import "context"

type Pipeline struct {
	stages []Stage
}

func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{
		stages: stages,
	}
}

func (p *Pipeline) Execute(
	ctx context.Context,
	state *Context,
) error {

	for _, stage := range p.stages {

		if err := stage.Execute(ctx, state); err != nil {
			return err
		}

	}

	return nil
}
