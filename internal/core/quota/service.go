package quota

type Service struct {
	repo Repository

	resolver PrincipalResolver

	evaluator *Evaluator
}
