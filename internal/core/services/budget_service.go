package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/curefatih/afi/internal/core/domain"
)

var ErrBudgetExceeded = errors.New("financial quota usage limits breached for requested context checkpoint")

type BudgetRepository interface {
	GetLimit(ctx context.Context, scope domain.BudgetScope, targetID string) (*domain.BudgetLimit, error)
	IncrementUsage(ctx context.Context, scope domain.BudgetScope, targetID string, amount float64) error
}

type BudgetService struct {
	repo BudgetRepository
}

func NewBudgetService(repo BudgetRepository) *BudgetService {
	return &BudgetService{repo: repo}
}

// Check sequentially steps through limits. If any checkpoint returns a breach, the entire execution is rejected.
func (s *BudgetService) Check(ctx context.Context, meta domain.RequestContext) error {
	checkpoints := []struct {
		scope domain.BudgetScope
		id    string
	}{
		{domain.ScopeOrg, meta.OrganizationID},
		{domain.ScopeTeam, meta.TeamID},
		{domain.ScopeProject, meta.ProjectID},
		{domain.ScopeKey, meta.APIKeyHash},
	}

	if meta.UserID != "" {
		checkpoints = append(checkpoints, struct {
			scope domain.BudgetScope
			id    string
		}{domain.ScopeUser, meta.UserID})
	}

	for _, cp := range checkpoints {
		if cp.id == "" {
			continue
		}
		limit, err := s.repo.GetLimit(ctx, cp.scope, cp.id)
		if err != nil {
			// If no explicit budget configured for this entity, consider it unlimited/pass through
			continue
		}

		if limit.UsedCost >= limit.MaxCost {
			return fmt.Errorf("%w: barrier triggered at %s scope level", ErrBudgetExceeded, cp.scope)
		}
	}

	return nil
}

// CommitUsage flushes multi-checkpoint increments back to the storage cluster.
func (s *BudgetService) CommitUsage(ctx context.Context, meta domain.RequestContext, usage domain.TokenUsage) error {
	checkpoints := []struct {
		scope domain.BudgetScope
		id    string
	}{
		{domain.ScopeOrg, meta.OrganizationID},
		{domain.ScopeTeam, meta.TeamID},
		{domain.ScopeProject, meta.ProjectID},
		{domain.ScopeKey, meta.APIKeyHash},
	}

	if meta.UserID != "" {
		checkpoints = append(checkpoints, struct {
			scope domain.BudgetScope
			id    string
		}{domain.ScopeUser, meta.UserID})
	}

	for _, cp := range checkpoints {
		if cp.id == "" {
			continue
		}
		// Distributed Redis lock or Atomic INCRBY float should be executed inside this repo call
		_ = s.repo.IncrementUsage(ctx, cp.scope, cp.id, usage.EstimatedCost)
	}

	return nil
}
