package gatewayconfig

import "context"

// MembershipReader resolves whether a quota scope belongs to an organization.
type MembershipReader interface {
	ProjectBelongsToOrg(ctx context.Context, projectID, orgID string) error
	TeamBelongsToOrg(ctx context.Context, teamID, orgID string) error
	UserIsOrgMember(ctx context.Context, userID, orgID string) error
	APIKeyBelongsToOrg(ctx context.Context, keyID, orgID string) error
}

// QuotaRepository persists write-model quotas.
type QuotaRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]Quota, error)
	Insert(ctx context.Context, q Quota) error
	UpdateLimit(ctx context.Context, quotaID string, limitValue int64) (*Quota, error)
	Delete(ctx context.Context, quotaID string) error
	OrgID(ctx context.Context, quotaID string) (string, error)
}
