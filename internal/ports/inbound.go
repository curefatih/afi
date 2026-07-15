package ports

import (
	"context"

	"github.com/curefatih/afi/internal/core/domain"
)

// LLMGatewayUseCase defines the primary orchestrator actions for passing requests
// through validation, budgeting, routing, and vendor mutations.
type LLMGatewayUseCase interface {
	ExecuteUnary(ctx context.Context, req *domain.InternalRequest) (*domain.InternalResponse, error)
	ExecuteStream(ctx context.Context, req *domain.InternalRequest) (<-chan domain.StreamChunk, <-chan error)
}

// TenantUseCase handles administration, onboarding, and access hierarchy validation.
type TenantUseCase interface {
	CreateOrganization(ctx context.Context, name string) (*domain.Organization, error)
	CreateTeam(ctx context.Context, orgID string, name string) (*domain.Team, error)
	CreateProject(ctx context.Context, teamID string, name string) (*domain.Project, error)
	AssignMembership(ctx context.Context, userID string, targetID string, role string) error
}

// AuthUseCase manages the creation and validation lifecycle of access credentials.
type AuthUseCase interface {
	IssueAPIKey(ctx context.Context, keyType domain.APIKeyType, targetID string) (string, error) // Returns raw key string once
	AuthenticateKey(ctx context.Context, rawKey string) (*domain.RequestContext, error)
}

type PlatformUserUseCase interface {
	RegisterPlatformUser(ctx context.Context, email string, password string) (*domain.PlatformUser, error)
	LoginPlatformWithEmailAndPassword(ctx context.Context, email string, password string) (*domain.Token, error)

	GetProfileByID(ctx context.Context, userID string) (*domain.PlatformUser, error)

	CreateCustomRole(ctx context.Context, role *domain.CustomRole) (*domain.CustomRole, error)
	AssignCustomRoleToUser(ctx context.Context, assignment *domain.UserAssignment) error

	ValidatePlatformUserToken(ctx context.Context, tokenStr string) (*domain.Token, error)
	GetUserPermissions(ctx context.Context, userID string, orgID string, projectID string) ([]domain.ActionPermission, error)

	GetUserOrganizations(ctx context.Context, userID string) ([]*domain.Organization, error)
}
