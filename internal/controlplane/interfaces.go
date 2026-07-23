package controlplane

import (
	"context"
)

// snapshotPublisher is implemented by *Seeder.
type snapshotPublisher interface {
	PublishSnapshot(ctx context.Context) error
}

// membershipChecker resolves org membership for authorization.
type membershipChecker interface {
	IsOrgMember(ctx context.Context, userID, orgID string) (bool, error)
	IsOrgAdmin(ctx context.Context, userID, orgID string) (bool, error)
	IsOrgOwner(ctx context.Context, userID, orgID string) (bool, error)
	GetTeamOrgID(ctx context.Context, teamID string) (string, error)
	CanAccessTeam(ctx context.Context, teamID, userID string) (bool, error)
	CanManageTeam(ctx context.Context, teamID, userID string) (bool, error)
	CanChangeTeamRoles(ctx context.Context, teamID, userID string) (bool, error)
	GetProjectOrgID(ctx context.Context, projectID string) (string, error)
	GetProviderOrgID(ctx context.Context, providerID string) (string, error)
	GetRouteOrgID(ctx context.Context, routeID string) (string, error)
	GetQuotaOrgID(ctx context.Context, quotaID string) (string, error)
	GetPolicyOrgID(ctx context.Context, policyID string) (string, error)
	GetWasmHookOrgID(ctx context.Context, hookID string) (string, error)
	GetMCPBackendOrgID(ctx context.Context, backendID string) (string, error)
	GetAPIKeyOrgID(ctx context.Context, keyID string) (string, error)
	GetCredentialOrgID(ctx context.Context, credentialID string) (string, error)
	GetCredentialAssignmentOrgID(ctx context.Context, assignmentID string) (string, error)
}

// platformAPI is the thin persistence surface still used directly by HTTP handlers
// for auth, org mail settings, and key/project authz helpers. App mutations go
// through platform.Service (s.app / s.config).
type platformAPI interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetOrganization(ctx context.Context, orgID string) (*Organization, error)
	SetOrgMailProvider(ctx context.Context, orgID, provider string) (*Organization, error)
	GetAPIKey(ctx context.Context, keyID string) (*APIKey, error)
	GetProjectOrgID(ctx context.Context, projectID string) (string, error)
}
