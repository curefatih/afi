package platform

import (
	"context"
	"time"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/credentials"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/curefatih/afi/internal/usage"
)

func (s *Service) ListOrganizationsForUser(ctx context.Context, userID string) ([]tenancy.Organization, error) {
	return s.API.ListOrganizationsForUser(ctx, userID)
}

func (s *Service) ListOrgMembers(ctx context.Context, orgID string) ([]tenancy.OrgMember, error) {
	return s.API.ListOrgMembers(ctx, orgID)
}

func (s *Service) ListOrgInvites(ctx context.Context, orgID string) ([]tenancy.OrgInvite, error) {
	return s.API.ListOrgInvites(ctx, orgID)
}

func (s *Service) PreviewOrgInvite(ctx context.Context, rawToken string) (*tenancy.InvitePreview, error) {
	return s.API.PreviewOrgInvite(ctx, rawToken)
}

func (s *Service) ListTeams(ctx context.Context, orgID, userID string) ([]tenancy.Team, error) {
	return s.API.ListTeams(ctx, orgID, userID)
}

func (s *Service) GetTeam(ctx context.Context, teamID string) (*tenancy.Team, error) {
	return s.API.GetTeam(ctx, teamID)
}

func (s *Service) ListTeamMembers(ctx context.Context, teamID string) ([]tenancy.TeamMember, error) {
	return s.API.ListTeamMembers(ctx, teamID)
}

func (s *Service) ListProjects(ctx context.Context, orgID, userID string) ([]tenancy.Project, error) {
	return s.API.ListProjects(ctx, orgID, userID)
}

func (s *Service) ListAPIKeys(ctx context.Context, projectID string) ([]access.APIKey, error) {
	return s.API.ListAPIKeys(ctx, projectID)
}

// ListVisibleOrgAPIKeys returns org keys filtered for the viewer.
// Admins see all keys; members see service accounts plus their own personal keys.
func (s *Service) ListVisibleOrgAPIKeys(ctx context.Context, orgID, viewerUserID string, viewerIsAdmin bool) ([]access.APIKey, error) {
	keys, err := s.API.ListOrgAPIKeys(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if viewerIsAdmin {
		return keys, nil
	}
	filtered := make([]access.APIKey, 0, len(keys))
	for _, k := range keys {
		if k.Kind == snapshot.KeyKindServiceAccount || k.OwnerUserID == viewerUserID {
			filtered = append(filtered, k)
		}
	}
	return filtered, nil
}

func (s *Service) ListProviders(ctx context.Context, orgID string) ([]gatewayconfig.Provider, error) {
	return s.API.ListProviders(ctx, orgID)
}

func (s *Service) ListProviderHealth(ctx context.Context, orgID string, from, to time.Time) ([]usage.ProviderHealth, error) {
	return s.API.ListProviderHealth(ctx, orgID, from, to)
}

func (s *Service) ListRoutes(ctx context.Context, orgID string) ([]gatewayconfig.Route, error) {
	return s.API.ListRoutes(ctx, orgID)
}

func (s *Service) ListUsage(ctx context.Context, orgID string, f usage.Filter) ([]usage.Record, error) {
	return s.API.ListUsage(ctx, orgID, f)
}

func (s *Service) SummarizeUsage(ctx context.Context, orgID string, f usage.Filter) ([]usage.SummaryBucket, error) {
	return s.API.SummarizeUsage(ctx, orgID, f)
}

func (s *Service) ListQuotas(ctx context.Context, orgID string) ([]gatewayconfig.Quota, error) {
	return s.API.ListQuotas(ctx, orgID)
}

func (s *Service) ListPolicies(ctx context.Context, orgID string) ([]gatewayconfig.RequestPolicy, error) {
	return s.API.ListPolicies(ctx, orgID)
}

func (s *Service) ListWasmHooks(ctx context.Context, orgID string) ([]gatewayconfig.WasmHook, error) {
	return s.API.ListWasmHooks(ctx, orgID)
}

func (s *Service) ListCredentials(ctx context.Context, orgID string) ([]credentials.Credential, error) {
	return s.API.ListCredentials(ctx, orgID)
}

func (s *Service) ListCredentialAssignments(ctx context.Context, orgID string) ([]credentials.Assignment, error) {
	return s.API.ListCredentialAssignments(ctx, orgID)
}
