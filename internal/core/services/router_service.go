package services

import (
	"context"

	"github.com/curefatih/afi/internal/core/domain"
)

type RouterRepository interface {
	Route(req *domain.InternalRequest) (domain.TargetDestination, error)
	AddRule(ctx context.Context, rule domain.RoutingRule) error
}

type RouterService struct {
	repo RouterRepository
}

func NewRouterService(repo RouterRepository) *RouterService {
	return &RouterService{repo: repo}
}

func (s *RouterService) Route(req *domain.InternalRequest) (domain.TargetDestination, error) {
	return s.repo.Route(req)
}

func (s *RouterService) AddRule(ctx context.Context, rule domain.RoutingRule) error {
	return s.repo.AddRule(ctx, rule)
}
