package access

import (
	"context"
	"time"
)

// timeNowUTC is overridden in tests.
var timeNowUTC = func() time.Time { return time.Now().UTC() }

// CreateAPIKey validates kind/project rules and persists the key.
func CreateAPIKey(
	ctx context.Context,
	repo APIKeyRepository,
	projects ProjectOrgChecker,
	envs EnvironmentProjectChecker,
	id, orgID, kind, ownerUserID, projectID, environmentID, name, rawKey string,
) (*APIKey, error) {
	k, err := NewAPIKey(id, orgID, kind, ownerUserID, projectID, environmentID, name, rawKey, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if projectID != "" {
		if err := projects.ProjectBelongsToOrg(ctx, projectID, orgID); err != nil {
			return nil, err
		}
	}
	if environmentID != "" {
		if err := envs.EnvironmentBelongsToProject(ctx, environmentID, projectID, orgID); err != nil {
			return nil, err
		}
	}
	if err := repo.Insert(ctx, *k, Hash(rawKey)); err != nil {
		return nil, err
	}
	return k, nil
}
