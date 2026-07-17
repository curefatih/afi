package gateway

type Pipeline struct {
	stages []Stage
}

func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{
		stages: stages,
	}
}

func (p *Pipeline) Execute(ctx *Context) error {

	for _, stage := range p.stages {

		if err := stage.Execute(ctx); err != nil {
			return err
		}

	}

	return nil
}