package quota

import "context"

type Service struct {
	repo Repository

	resolver PrincipalResolver

	evaluator Evaluator
}

func NewService(
	repo Repository,
	resolver PrincipalResolver,
	evaluator Evaluator,
) *Service {
	return &Service{
		repo:      repo,
		resolver:  resolver,
		evaluator: evaluator,
	}
}

func (s *Service) Check(
	ctx context.Context,
	req *Request,
) (*Decision, error) {

	targets := s.resolver.Scopes(*req.Principal)

	var quotas []Quota

	for _, target := range targets {

		items, err := s.repo.List(
			ctx,
			target.Scope,
			target.ID,
		)

		if err != nil {
			return nil, err
		}

		quotas = append(quotas, items...)
	}

	decision := s.evaluator.Evaluate(
		quotas,
		req.Usage.Items,
	)

	return &decision, nil
}

func (s *Service) Commit(
	ctx context.Context,
	req *CommitRequest,
) error {

	targets := s.resolver.Scopes(*req.Principal)

	for _, target := range targets {

		if err := s.repo.Commit(
			ctx,
			target.Scope,
			target.ID,
			req.Usage.Items,
		); err != nil {
			return err
		}

	}

	return nil
}
