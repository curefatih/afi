package services

import (
	"context"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/core/domain"
)

type TenantRepository interface {
	SaveOrganization(ctx context.Context, org *domain.Organization) error
	SaveTeam(ctx context.Context, team *domain.Team) error
	SaveProject(ctx context.Context, project *domain.Project) error
	SaveMembership(ctx context.Context, membership *domain.Membership) error
}

type TenantService struct {
	repo TenantRepository
}

func NewTenantService(repo TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

func (s *TenantService) CreateOrganization(ctx context.Context, name string) (*domain.Organization, error) {
	org := &domain.Organization{
		ID: fmt.Sprintf("org_%d", time.Now().UnixNano()), // Replace with a clean UUID engine
	}
	if err := s.repo.SaveOrganization(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *TenantService) CreateTeam(ctx context.Context, orgID string, name string) (*domain.Team, error) {
	team := &domain.Team{
		ID:    fmt.Sprintf("team_%d", time.Now().UnixNano()),
		OrgID: orgID,
	}
	if err := s.repo.SaveTeam(ctx, team); err != nil {
		return nil, err
	}
	return team, nil
}

func (s *TenantService) CreateProject(ctx context.Context, teamID string, name string) (*domain.Project, error) {
	project := &domain.Project{
		ID:     fmt.Sprintf("proj_%d", time.Now().UnixNano()),
		TeamID: teamID,
	}
	if err := s.repo.SaveProject(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *TenantService) AssignMembership(ctx context.Context, userID string, targetID string, role string) error {
	membership := &domain.Membership{
		UserID:   userID,
		TargetID: targetID,
		Role:     role,
	}
	return s.repo.SaveMembership(ctx, membership)
}
