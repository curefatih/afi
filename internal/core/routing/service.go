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
	req ResolveRequest,
) (*Decision, error) {

	route, err := s.repo.Find(
		ctx,
		req,
	)

	if err != nil {
		return nil, err
	}

	return s.selector.Select(
		ctx,
		*route,
	)
}
