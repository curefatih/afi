package platform

import (
	"context"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/credentials"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
)

// PublishSnapshot republishes the gateway snapshot and emits snapshot.published.
func (s *Service) PublishSnapshot(ctx context.Context) error {
	return s.publish(ctx, "published")
}

// Tenancy/invite/team commands persist then emit only — membership is not in the gateway snapshot.

func (s *Service) CreateOrganization(ctx context.Context, name, creatorUserID string) (*tenancy.Organization, error) {
	org, err := s.API.CreateOrganization(ctx, name, creatorUserID)
	if err != nil {
		return nil, err
	}
	s.emit(ctx, EventOrgCreated, org.ID, org.ID)
	return org, nil
}

func (s *Service) UpdateOrgMemberRole(ctx context.Context, orgID, actorUserID, targetUserID, role string) (*tenancy.OrgMember, error) {
	member, err := s.API.UpdateOrgMemberRole(ctx, orgID, actorUserID, targetUserID, role)
	if err != nil {
		return nil, err
	}
	s.emit(ctx, EventMemberRoleUpdated, targetUserID, orgID)
	return member, nil
}

func (s *Service) InviteOrgMember(ctx context.Context, orgID, email, invitedByUserID string) (*tenancy.InviteOutcome, string, error) {
	outcome, rawToken, err := s.API.InviteOrgMember(ctx, orgID, email, invitedByUserID)
	if err != nil {
		return nil, "", err
	}
	if outcome != nil && outcome.Status == "added" && outcome.Member != nil {
		s.emit(ctx, EventMemberAdded, outcome.Member.UserID, orgID)
	} else if outcome != nil && outcome.Invite != nil {
		s.emit(ctx, EventInviteCreated, outcome.Invite.ID, orgID)
	}
	return outcome, rawToken, nil
}

func (s *Service) RevokeOrgInvite(ctx context.Context, orgID, inviteID string) error {
	if err := s.API.RevokeOrgInvite(ctx, orgID, inviteID); err != nil {
		return err
	}
	s.emit(ctx, EventInviteRevoked, inviteID, orgID)
	return nil
}

func (s *Service) ResendOrgInvite(ctx context.Context, orgID, inviteID string) (*tenancy.OrgInvite, string, error) {
	inv, rawToken, err := s.API.ResendOrgInvite(ctx, orgID, inviteID)
	if err != nil {
		return nil, "", err
	}
	s.emit(ctx, EventInviteResent, inviteID, orgID)
	return inv, rawToken, nil
}

func (s *Service) AcceptOrgInvite(ctx context.Context, rawToken, name, passwordHash string) (*tenancy.OrgMember, *identity.User, error) {
	preview, err := s.API.PreviewOrgInvite(ctx, rawToken)
	if err != nil {
		return nil, nil, err
	}
	member, user, err := s.API.AcceptOrgInvite(ctx, rawToken, name, passwordHash)
	if err != nil {
		return nil, nil, err
	}
	orgID := ""
	if preview != nil {
		orgID = preview.OrganizationID
	}
	resourceID := ""
	if member != nil {
		resourceID = member.UserID
	}
	s.emit(ctx, EventInviteAccepted, resourceID, orgID)
	return member, user, nil
}

func (s *Service) CreateTeam(ctx context.Context, orgID, name, creatorUserID string) (*tenancy.Team, error) {
	team, err := s.API.CreateTeam(ctx, orgID, name, creatorUserID)
	if err != nil {
		return nil, err
	}
	s.emit(ctx, EventTeamCreated, team.ID, orgID)
	return team, nil
}

func (s *Service) AddTeamMember(ctx context.Context, teamID, userID string) (*tenancy.TeamMember, error) {
	member, err := s.API.AddTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	orgID := ""
	if team, tErr := s.API.GetTeam(ctx, teamID); tErr == nil && team != nil {
		orgID = team.OrganizationID
	}
	s.emit(ctx, EventTeamMemberAdded, userID, orgID)
	return member, nil
}

func (s *Service) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	orgID := ""
	if team, tErr := s.API.GetTeam(ctx, teamID); tErr == nil && team != nil {
		orgID = team.OrganizationID
	}
	if err := s.API.RemoveTeamMember(ctx, teamID, userID); err != nil {
		return err
	}
	s.emit(ctx, EventTeamMemberRemoved, userID, orgID)
	return nil
}

func (s *Service) CreateProject(ctx context.Context, orgID, teamID, name string) (*tenancy.Project, error) {
	p, err := s.API.CreateProject(ctx, orgID, teamID, name)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "created"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventProjectCreated, p.ID, orgID)
	return p, nil
}

func (s *Service) CreateAPIKey(ctx context.Context, orgID, kind, ownerUserID, projectID, name, rawKey string) (*access.APIKey, error) {
	k, err := s.API.CreateAPIKey(ctx, orgID, kind, ownerUserID, projectID, name, rawKey)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "created"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventAPIKeyCreated, k.ID, orgID)
	return k, nil
}

func (s *Service) DeleteAPIKey(ctx context.Context, keyID string) error {
	if err := s.API.DeleteAPIKey(ctx, keyID); err != nil {
		return err
	}
	if err := s.publish(ctx, "deleted"); err != nil {
		return err
	}
	s.emit(ctx, EventAPIKeyDeleted, keyID, "")
	return nil
}

func (s *Service) CreateProvider(ctx context.Context, orgID, name, typ, baseURL, apiKeyEnv string, caps snapshot.ProviderCapabilities) (*gatewayconfig.Provider, error) {
	p, err := s.API.CreateProvider(ctx, orgID, name, typ, baseURL, apiKeyEnv, caps)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "created"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventProviderCreated, p.ID, orgID)
	return p, nil
}

func (s *Service) UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*gatewayconfig.Provider, error) {
	p, err := s.API.UpdateProvider(ctx, providerID, name, baseURL, apiKeyEnv)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "updated"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventProviderUpdated, providerID, p.OrganizationID)
	return p, nil
}

func (s *Service) DeleteProvider(ctx context.Context, providerID string) error {
	if err := s.API.DeleteProvider(ctx, providerID); err != nil {
		return err
	}
	if err := s.publish(ctx, "deleted"); err != nil {
		return err
	}
	s.emit(ctx, EventProviderDeleted, providerID, "")
	return nil
}

func (s *Service) CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error) {
	r, err := s.API.CreateRoute(ctx, orgID, model, providerID, targetModel, fallbacks)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "created"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventRouteCreated, r.ID, orgID)
	return r, nil
}

func (s *Service) UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error) {
	r, err := s.API.UpdateRoute(ctx, routeID, model, providerID, targetModel, fallbacks)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "updated"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventRouteUpdated, routeID, r.OrganizationID)
	return r, nil
}

func (s *Service) DeleteRoute(ctx context.Context, routeID string) error {
	if err := s.API.DeleteRoute(ctx, routeID); err != nil {
		return err
	}
	if err := s.publish(ctx, "deleted"); err != nil {
		return err
	}
	s.emit(ctx, EventRouteDeleted, routeID, "")
	return nil
}

func (s *Service) CreateQuota(ctx context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*gatewayconfig.Quota, error) {
	q, err := s.API.CreateQuota(ctx, orgID, scopeType, scopeID, metric, limitValue, window)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "created"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventQuotaCreated, q.ID, orgID)
	return q, nil
}

func (s *Service) UpdateQuota(ctx context.Context, quotaID string, limitValue int64) (*gatewayconfig.Quota, error) {
	q, err := s.API.UpdateQuota(ctx, quotaID, limitValue)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "updated"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventQuotaUpdated, quotaID, q.OrganizationID)
	return q, nil
}

func (s *Service) DeleteQuota(ctx context.Context, quotaID string) error {
	if err := s.API.DeleteQuota(ctx, quotaID); err != nil {
		return err
	}
	if err := s.publish(ctx, "deleted"); err != nil {
		return err
	}
	s.emit(ctx, EventQuotaDeleted, quotaID, "")
	return nil
}

func (s *Service) CreatePolicy(ctx context.Context, orgID, name, expression string, enabled bool, priority int) (*gatewayconfig.RequestPolicy, error) {
	p, err := s.API.CreatePolicy(ctx, orgID, name, expression, enabled, priority)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "created"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventPolicyCreated, p.ID, orgID)
	return p, nil
}

func (s *Service) UpdatePolicy(ctx context.Context, policyID string, name, expression *string, enabled *bool, priority *int) (*gatewayconfig.RequestPolicy, error) {
	p, err := s.API.UpdatePolicy(ctx, policyID, name, expression, enabled, priority)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "updated"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventPolicyUpdated, policyID, p.OrganizationID)
	return p, nil
}

func (s *Service) DeletePolicy(ctx context.Context, policyID string) error {
	if err := s.API.DeletePolicy(ctx, policyID); err != nil {
		return err
	}
	if err := s.publish(ctx, "deleted"); err != nil {
		return err
	}
	s.emit(ctx, EventPolicyDeleted, policyID, "")
	return nil
}

func (s *Service) CreateCredential(ctx context.Context, orgID, name, providerType, storageKind, secretRef, secretValue string) (*credentials.Credential, error) {
	c, err := s.API.CreateCredential(ctx, orgID, name, providerType, storageKind, secretRef, secretValue)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "created"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventCredentialCreated, c.ID, orgID)
	return c, nil
}

func (s *Service) UpdateCredential(ctx context.Context, credentialID, name, status string) (*credentials.Credential, error) {
	c, err := s.API.UpdateCredential(ctx, credentialID, name, status)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "updated"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventCredentialUpdated, credentialID, c.OrganizationID)
	return c, nil
}

func (s *Service) RotateCredential(ctx context.Context, credentialID, secretRef, secretValue string) (*credentials.Credential, error) {
	c, err := s.API.RotateCredential(ctx, credentialID, secretRef, secretValue)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "updated"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventCredentialRotated, credentialID, c.OrganizationID)
	return c, nil
}

func (s *Service) DeleteCredential(ctx context.Context, credentialID string) error {
	if err := s.API.DeleteCredential(ctx, credentialID); err != nil {
		return err
	}
	if err := s.publish(ctx, "deleted"); err != nil {
		return err
	}
	s.emit(ctx, EventCredentialDeleted, credentialID, "")
	return nil
}

func (s *Service) AssignCredential(ctx context.Context, credentialID, scopeType, scopeID, createdBy string) (*credentials.Assignment, error) {
	a, err := s.API.AssignCredential(ctx, credentialID, scopeType, scopeID, createdBy)
	if err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "created"); err != nil {
		return nil, err
	}
	s.emit(ctx, EventCredentialAssigned, a.ID, a.OrganizationID)
	return a, nil
}

func (s *Service) DeleteCredentialAssignment(ctx context.Context, assignmentID string) error {
	if err := s.API.DeleteCredentialAssignment(ctx, assignmentID); err != nil {
		return err
	}
	if err := s.publish(ctx, "deleted"); err != nil {
		return err
	}
	s.emit(ctx, EventCredentialUnassigned, assignmentID, "")
	return nil
}
