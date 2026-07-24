package access

import "context"

// ProjectOrgChecker verifies a project belongs to an organization.
type ProjectOrgChecker interface {
	ProjectBelongsToOrg(ctx context.Context, projectID, orgID string) error
}

// EnvironmentProjectChecker verifies an environment belongs to a project in an org.
type EnvironmentProjectChecker interface {
	EnvironmentBelongsToProject(ctx context.Context, environmentID, projectID, orgID string) error
}

// APIKeyRepository persists write-model API keys.
type APIKeyRepository interface {
	ListByProject(ctx context.Context, projectID string) ([]APIKey, error)
	ListByOrg(ctx context.Context, orgID string) ([]APIKey, error)
	Get(ctx context.Context, keyID string) (*APIKey, error)
	Insert(ctx context.Context, key APIKey, keyHash string) error
	Delete(ctx context.Context, keyID string) error
	OrgID(ctx context.Context, keyID string) (string, error)
}
