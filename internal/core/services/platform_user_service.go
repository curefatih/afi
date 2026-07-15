package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
	"golang.org/x/crypto/bcrypt"
)

// PlatformUserRepository defines the persistence operations for platform user management.
type PlatformUserRepository interface {
	SaveUser(ctx context.Context, user *domain.PlatformUser) error
	GetUserByEmail(ctx context.Context, email string) (*domain.PlatformUser, error)

	SaveCustomRole(ctx context.Context, role *domain.CustomRole) error
	SaveRoleAssignment(ctx context.Context, assignment *domain.UserAssignment) error

	GetUserPermissions(ctx context.Context, userID string, orgID string, projectID string) ([]domain.ActionPermission, error)
	GetUserByID(ctx context.Context, userID string) (*domain.PlatformUser, error)

	GetUserOrganizations(ctx context.Context, userID string) ([]*domain.Organization, error)
	GetUserOrganizationProjects(ctx context.Context, userID string, orgID string) ([]*domain.Project, error)
	GetUserProjects(ctx context.Context, userID string) ([]*domain.Project, error)
}

type PlatformUserService struct {
	repo     PlatformUserRepository
	tokenSvc ports.PlatformTokenService
}

// GetProfileByID implements ports.PlatformUserUseCase.
func (s *PlatformUserService) GetProfileByID(ctx context.Context, userID string) (*domain.PlatformUser, error) {
	// Simply delegate database fetching downstream to your PlatformUserRepository
	// If needed, your repository can support a standard GetUserByID method.
	return s.repo.GetUserByID(ctx, userID)
}

var _ ports.PlatformUserUseCase = &PlatformUserService{}

func NewPlatformUserService(repo PlatformUserRepository, tokenSvc ports.PlatformTokenService) *PlatformUserService {
	return &PlatformUserService{repo: repo, tokenSvc: tokenSvc}
}

// ValidatePlatformUserToken implements ports.PlatformUserUseCase.
func (s *PlatformUserService) ValidatePlatformUserToken(ctx context.Context, tokenStr string) (*domain.Token, error) {
	_, err := s.tokenSvc.ValidateToken(ctx, tokenStr)
	if err != nil {
		return nil, err
	}
	return &domain.Token{Token: tokenStr}, nil
}

// GeneratePlatformUserToken implements ports.PlatformUserUseCase.
func (s *PlatformUserService) GeneratePlatformUserToken(ctx context.Context, user *domain.PlatformUser) (*domain.Token, error) {
	token, err := s.tokenSvc.GenerateToken(ctx, user)
	if err != nil {
		return nil, err
	}
	return &domain.Token{Token: token}, nil
}

// RegisterPlatformUser handles password hashing and user creation
func (s *PlatformUserService) RegisterPlatformUser(ctx context.Context, email string, password string) (*domain.PlatformUser, error) {
	// 1. Hash the incoming raw password securely
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to process password credentials: %w", err)
	}

	user := &domain.PlatformUser{
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	// 2. Persist to storage layer
	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// LoginPlatformWithEmailAndPassword validates password matching hashes and issues a session token
func (s *PlatformUserService) LoginPlatformWithEmailAndPassword(ctx context.Context, email string, password string) (*domain.Token, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("invalid platform user credentials")
	}

	// Compare cryptographically secure bcrypt hashes
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid platform user credentials")
	}

	// Delegate stateless string creation up to the out-of-core crypto wrapper
	token, err := s.tokenSvc.GenerateToken(ctx, user)
	if err != nil {
		return nil, err
	}
	return &domain.Token{Token: token}, nil
}

// CreateCustomRole commits a granular policy schema rule layout definition
func (s *PlatformUserService) CreateCustomRole(ctx context.Context, role *domain.CustomRole) (*domain.CustomRole, error) {
	if role.Name == "" || len(role.Permissions) == 0 {
		return nil, errors.New("invalid custom role configuration parameters: name and permissions required")
	}
	if err := s.repo.SaveCustomRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// AssignCustomRoleToUser maps a user identification profile directly to a policy layout
func (s *PlatformUserService) AssignCustomRoleToUser(ctx context.Context, assignment *domain.UserAssignment) error {
	if assignment.UserID == "" || assignment.RoleName == "" {
		return errors.New("invalid role assignment matrix: userID and role reference target required")
	}
	return s.repo.SaveRoleAssignment(ctx, assignment)
}

// GetUserPermissions gathers all explicit and custom permissions a user holds
func (s *PlatformUserService) GetUserPermissions(ctx context.Context, userID string, orgID string, projectID string) ([]domain.ActionPermission, error) {
	if userID == "" || orgID == "" {
		return nil, errors.New("missing minimum verification context identity: userID and orgID required")
	}
	return s.repo.GetUserPermissions(ctx, userID, orgID, projectID)
}

// GetUserOrganizations implements ports.PlatformUserUseCase.
func (s *PlatformUserService) GetUserOrganizations(ctx context.Context, userID string) ([]*domain.Organization, error) {
	if userID == "" {
		return nil, errors.New("missing userID for organization retrieval")
	}
	return s.repo.GetUserOrganizations(ctx, userID)
}

// GetUserOrganizations implements ports.PlatformUserUseCase.
func (s *PlatformUserService) GetUserProjects(ctx context.Context, userID string) ([]*domain.Project, error) {
	if userID == "" {
		return nil, errors.New("missing userID for organization retrieval")
	}
	return s.repo.GetUserProjects(ctx, userID)
}

// GetUserOrganizations implements ports.PlatformUserUseCase.
func (s *PlatformUserService) GetUserOrganizationProjects(ctx context.Context, userID string, orgID string) ([]*domain.Project, error) {
	if userID == "" {
		return nil, errors.New("missing userID for organization retrieval")
	}
	return s.repo.GetUserOrganizationProjects(ctx, userID, orgID)
}
