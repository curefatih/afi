package gatewayconfig

import (
	"context"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

// AssertScopeInOrg verifies the scope entity belongs to the organization.
func AssertScopeInOrg(ctx context.Context, orgID, scopeType, scopeID string, m MembershipReader) error {
	if err := ValidateScopeType(scopeType); err != nil {
		return err
	}
	if scopeID == "" {
		return fmt.Errorf("%w: scope_id is required", kernel.ErrInvalidRequest)
	}
	switch scopeType {
	case snapshot.ScopeOrganization:
		if scopeID != orgID {
			return fmt.Errorf("%w: organization scope_id must match organization", kernel.ErrInvalidRequest)
		}
		return nil
	case snapshot.ScopeTeam:
		return m.TeamBelongsToOrg(ctx, scopeID, orgID)
	case snapshot.ScopeProject:
		return m.ProjectBelongsToOrg(ctx, scopeID, orgID)
	case snapshot.ScopeUser:
		return m.UserIsOrgMember(ctx, scopeID, orgID)
	case snapshot.ScopeAPIKey:
		return m.APIKeyBelongsToOrg(ctx, scopeID, orgID)
	default:
		return fmt.Errorf("%w: invalid scope_type %q", kernel.ErrInvalidRequest, scopeType)
	}
}

// CreateQuota validates fields + membership, then persists the quota.
func CreateQuota(
	ctx context.Context,
	repo QuotaRepository,
	members MembershipReader,
	id string,
	orgID, scopeType, scopeID, metric string,
	limitValue int64,
	window string,
) (*Quota, error) {
	window = NormalizeWindow(window)
	q, err := NewQuota(id, orgID, scopeType, scopeID, metric, limitValue, window, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := AssertScopeInOrg(ctx, orgID, scopeType, scopeID, members); err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *q); err != nil {
		return nil, err
	}
	return q, nil
}

// timeNowUTC is overridden in tests.
var timeNowUTC = func() time.Time { return time.Now().UTC() }
