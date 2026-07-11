package services

import (
	"context"

	"github.com/curefatih/afi/internal/core/domain"
)

// PlatformUserRepository defines the persistence operations for platform user management.
type PlatformUserRepository interface {
	SaveUser(ctx context.Context, user *domain.PlatformUser) error
	GetUserByEmail(ctx context.Context, email string) (*domain.PlatformUser, error)

	SaveCustomRole(ctx context.Context, role *domain.CustomRole) error
	SaveRoleAssignment(ctx context.Context, assignment *domain.UserAssignment) error

	GetUserPermissions(ctx context.Context, userID string, orgID string, projectID string) ([]domain.ActionPermission, error)
}

type PlatformUserService struct {
	repo PlatformUserRepository
}

func NewPlatformUserService(repo PlatformUserRepository) *PlatformUserService {
	return &PlatformUserService{repo: repo}
}

func (s *PlatformUserService) GenerateToken(ctx context.Context, user *domain.PlatformUser) (string, error) {
	// TODO: Implement token generation
	return "", nil
}

func (s *PlatformUserService) ValidateToken(ctx context.Context, tokenStr string) (string, error) {
	// TODO: Implement token validation
	return "", nil
}

func (s *PlatformUserService) GetUserPermissions(ctx context.Context, userID string, orgID string, projectID string) ([]domain.ActionPermission, error) {
	return s.repo.GetUserPermissions(ctx, userID, orgID, projectID)
}
