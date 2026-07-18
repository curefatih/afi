package model

import "context"

type Service struct {
	repo Repository

	registry Registry

	validator Validator

	selector Selector
}

func NewService(repo Repository, registry Registry, validator Validator, selector Selector) *Service {
	return &Service{
		repo:      repo,
		registry:  registry,
		validator: validator,
		selector:  selector,
	}
}

func (s *Service) Resolve(
	ctx context.Context,
	name string,
) (*Model, error) {
	return s.selector.Resolve(ctx, name)
}
