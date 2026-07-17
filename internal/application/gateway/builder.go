package gateway

type Builder struct {
	stages []Stage
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Add(stage Stage) *Builder {

	b.stages = append(b.stages, stage)

	return b
}

func (b *Builder) Build() *Pipeline {

	return NewPipeline(
		b.stages...,
	)
}