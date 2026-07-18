package routing

import "context"

type Service struct {
	repo Repository

	selector Selector
}

func NewService(
	repo Repository,
	selector Selector,
) *Service {
	return &Service{
		repo:     repo,
		selector: selector,
	}
}

func (s *Service) Resolve(
	ctx context.Context,
	request ResolveRequest,
) (*Decision, error) {

	candidates, err := s.repo.FindCandidates(
		ctx,
		request,
	)

	if err != nil {
		return nil, err
	}

	return s.selector.Select(
		ctx,
		candidates,
	)
}
