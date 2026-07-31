package platform

import (
	"context"
	"encoding/json"
	"time"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/audit"
	"github.com/curefatih/afi/internal/credentials"
	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/curefatih/afi/internal/usage"
)

// SnapshotPublisher publishes compiled gateway snapshots after config changes.
type SnapshotPublisher interface {
	PublishSnapshot(ctx context.Context) error
	// PublishRegionSnapshots puts the global snapshot and only the listed regions' object-store blobs.
	PublishRegionSnapshots(ctx context.Context, regionIDs ...string) error
}

// ConfigAPI is the persistence surface for platform reads and mutations.
type ConfigAPI interface {
	ListOrganizationsForUser(ctx context.Context, userID string) ([]tenancy.Organization, error)
	CreateOrganization(ctx context.Context, name, creatorUserID string) (*tenancy.Organization, error)
	ListOrgMembers(ctx context.Context, orgID string) ([]tenancy.OrgMember, error)
	UpdateOrgMemberRole(ctx context.Context, orgID, actorUserID, targetUserID, role string) (*tenancy.OrgMember, error)
	InviteOrgMember(ctx context.Context, orgID, email, invitedByUserID string) (*tenancy.InviteOutcome, string, error)
	ListOrgInvites(ctx context.Context, orgID string) ([]tenancy.OrgInvite, error)
	RevokeOrgInvite(ctx context.Context, orgID, inviteID string) error
	ResendOrgInvite(ctx context.Context, orgID, inviteID string) (*tenancy.OrgInvite, string, error)
	PreviewOrgInvite(ctx context.Context, rawToken string) (*tenancy.InvitePreview, error)
	AcceptOrgInvite(ctx context.Context, rawToken, name, passwordHash string) (*tenancy.OrgMember, *identity.User, error)

	ListTeams(ctx context.Context, orgID, userID string) ([]tenancy.Team, error)
	CreateTeam(ctx context.Context, orgID, name, creatorUserID string) (*tenancy.Team, error)
	GetTeam(ctx context.Context, teamID string) (*tenancy.Team, error)
	ListTeamMembers(ctx context.Context, teamID string) ([]tenancy.TeamMember, error)
	AddTeamMember(ctx context.Context, teamID, userID string) (*tenancy.TeamMember, error)
	UpdateTeamMemberRole(ctx context.Context, teamID, actorUserID, targetUserID, role string) (*tenancy.TeamMember, error)
	RemoveTeamMember(ctx context.Context, teamID, userID string) error
	ListProjects(ctx context.Context, orgID, userID string) ([]tenancy.Project, error)
	CreateProject(ctx context.Context, orgID, teamID, name string) (*tenancy.Project, error)

	ListEnvironments(ctx context.Context, projectID string) ([]tenancy.Environment, error)
	GetEnvironment(ctx context.Context, environmentID string) (*tenancy.Environment, error)
	CreateEnvironment(ctx context.Context, orgID, projectID, name, slug string) (*tenancy.Environment, error)
	DeleteEnvironment(ctx context.Context, environmentID string) error
	GetEnvironmentOrgID(ctx context.Context, environmentID string) (string, error)

	ListAPIKeys(ctx context.Context, projectID string) ([]access.APIKey, error)
	ListOrgAPIKeys(ctx context.Context, orgID string) ([]access.APIKey, error)
	CreateAPIKey(ctx context.Context, orgID, kind, ownerUserID, projectID, environmentID, name, rawKey string) (*access.APIKey, error)
	GetAPIKeyOrgID(ctx context.Context, keyID string) (string, error)
	DeleteAPIKey(ctx context.Context, keyID string) error
	ListSigningKeys(ctx context.Context, orgID string) ([]access.SigningKey, error)
	CreateSigningKey(ctx context.Context, orgID, keyID, projectID, environmentID, name, algorithm, publicKeyPEM string) (*access.SigningKey, error)
	UpdateSigningKey(ctx context.Context, id, name, status string) (*access.SigningKey, error)
	RotateSigningKey(ctx context.Context, id, publicKeyPEM string) (*access.SigningKey, error)
	GetSigningKeyOrgID(ctx context.Context, id string) (string, error)
	DeleteSigningKey(ctx context.Context, id string) error

	ListProviders(ctx context.Context, orgID string) ([]gatewayconfig.Provider, error)
	ListProviderHealth(ctx context.Context, orgID string, from, to time.Time) ([]usage.ProviderHealth, error)
	CreateProvider(ctx context.Context, orgID, name, typ, baseURL, apiKeyEnv string, caps snapshot.ProviderCapabilities, config json.RawMessage) (*gatewayconfig.Provider, error)
	UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string, config *json.RawMessage) (*gatewayconfig.Provider, error)
	GetProviderOrgID(ctx context.Context, providerID string) (string, error)
	DeleteProvider(ctx context.Context, providerID string) error

	ListRoutes(ctx context.Context, orgID string) ([]gatewayconfig.Route, error)
	CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback, retry *gatewayconfig.RetryConfig, strategy string, weight int) (*gatewayconfig.Route, error)
	UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback, retry *gatewayconfig.RetryConfig, strategy string, weight int) (*gatewayconfig.Route, error)
	GetRouteOrgID(ctx context.Context, routeID string) (string, error)
	DeleteRoute(ctx context.Context, routeID string) error
	GetOrgDefaultRetry(ctx context.Context, orgID string) (*gatewayconfig.RetryConfig, error)
	SetOrgDefaultRetry(ctx context.Context, orgID string, retry *gatewayconfig.RetryConfig) error
	GetOrgObjectStore(ctx context.Context, orgID string) (*gatewayconfig.ObjectStoreConfig, error)
	SetOrgObjectStore(ctx context.Context, orgID string, cfg *gatewayconfig.ObjectStoreConfig) error

	ListUsage(ctx context.Context, orgID string, f usage.Filter) ([]usage.Record, error)
	SummarizeUsage(ctx context.Context, orgID string, f usage.Filter) ([]usage.SummaryBucket, error)
	ListAudit(ctx context.Context, orgID string, f audit.Filter) ([]audit.Record, error)

	ListQuotas(ctx context.Context, orgID string) ([]gatewayconfig.Quota, error)
	CreateQuota(ctx context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*gatewayconfig.Quota, error)
	UpdateQuota(ctx context.Context, quotaID string, limitValue int64) (*gatewayconfig.Quota, error)
	GetQuotaOrgID(ctx context.Context, quotaID string) (string, error)
	DeleteQuota(ctx context.Context, quotaID string) error

	ListPolicies(ctx context.Context, orgID string) ([]gatewayconfig.RequestPolicy, error)
	CreatePolicy(ctx context.Context, orgID, name, expression string, actions []gatewayconfig.PolicyAction, enabled bool, priority int) (*gatewayconfig.RequestPolicy, error)
	UpdatePolicy(ctx context.Context, policyID string, name, expression *string, actions []gatewayconfig.PolicyAction, enabled *bool, priority *int) (*gatewayconfig.RequestPolicy, error)
	ReorderPolicies(ctx context.Context, orgID string, items []gatewayconfig.PolicyPriorityUpdate) error
	GetPolicyOrgID(ctx context.Context, policyID string) (string, error)
	DeletePolicy(ctx context.Context, policyID string) error

	ListWasmHooks(ctx context.Context, orgID string) ([]gatewayconfig.WasmHook, error)
	CreateWasmHook(ctx context.Context, orgID, name, phase, moduleURI, digest string, enabled bool, priority int, config []byte) (*gatewayconfig.WasmHook, error)
	UpdateWasmHook(ctx context.Context, id string, name, phase, moduleURI, digest *string, enabled *bool, priority *int, config []byte) (*gatewayconfig.WasmHook, error)
	GetWasmHookOrgID(ctx context.Context, id string) (string, error)
	DeleteWasmHook(ctx context.Context, id string) error

	ListMCPBackends(ctx context.Context, orgID string) ([]gatewayconfig.MCPBackend, error)
	CreateMCPBackend(ctx context.Context, orgID, alias, name, baseURL, apiKeyEnv string, methodAllowlist []byte, enabled bool) (*gatewayconfig.MCPBackend, error)
	UpdateMCPBackend(ctx context.Context, id string, alias, name, baseURL, apiKeyEnv *string, methodAllowlist []byte, enabled *bool) (*gatewayconfig.MCPBackend, error)
	GetMCPBackendOrgID(ctx context.Context, id string) (string, error)
	DeleteMCPBackend(ctx context.Context, id string) error

	ListA2AAgents(ctx context.Context, orgID string) ([]gatewayconfig.A2AAgent, error)
	CreateA2AAgent(ctx context.Context, orgID, alias, name, upstreamURL, cardURL, apiKeyEnv, authScheme string, cardCache []byte, enabled bool) (*gatewayconfig.A2AAgent, error)
	UpdateA2AAgent(ctx context.Context, id string, alias, name, upstreamURL, cardURL, apiKeyEnv, authScheme *string, cardCache []byte, enabled *bool) (*gatewayconfig.A2AAgent, error)
	GetA2AAgentOrgID(ctx context.Context, id string) (string, error)
	DeleteA2AAgent(ctx context.Context, id string) error

	ListCredentials(ctx context.Context, orgID string) ([]credentials.Credential, error)
	CreateCredential(ctx context.Context, orgID, name, providerType, storageKind, secretRef, secretValue string) (*credentials.Credential, error)
	UpdateCredential(ctx context.Context, credentialID, name, status string) (*credentials.Credential, error)
	RotateCredential(ctx context.Context, credentialID, secretRef, secretValue string) (*credentials.Credential, error)
	GetCredentialOrgID(ctx context.Context, credentialID string) (string, error)
	DeleteCredential(ctx context.Context, credentialID string) error
	ListCredentialAssignments(ctx context.Context, orgID string) ([]credentials.Assignment, error)
	AssignCredential(ctx context.Context, credentialID, scopeType, scopeID, createdBy string) (*credentials.Assignment, error)
	GetCredentialAssignmentOrgID(ctx context.Context, assignmentID string) (string, error)
	DeleteCredentialAssignment(ctx context.Context, assignmentID string) error

	ListRegions(ctx context.Context) ([]regions.Region, error)
	GetRegion(ctx context.Context, regionID string) (*regions.Region, error)
	CreateRegion(ctx context.Context, slug, name string) (*regions.Region, error)
	UpdateRegion(ctx context.Context, regionID, name, status string) (*regions.Region, error)
	ListDeployments(ctx context.Context, regionID string) ([]regions.GatewayDeployment, error)
	GetDeployment(ctx context.Context, deploymentID string) (*regions.GatewayDeployment, error)
	RegisterDeployment(ctx context.Context, regionID, name, publicBaseURL string) (*regions.DeploymentWithToken, error)
	RotateDeploymentJoinToken(ctx context.Context, deploymentID string) (*regions.DeploymentWithToken, error)
	RecordDeploymentHeartbeat(ctx context.Context, deploymentID, joinToken string, snapVersion int64, build string) (*regions.GatewayDeployment, error)
	AuthenticateDeploymentJoinToken(ctx context.Context, rawToken string) (*regions.GatewayDeployment, error)

	ListRegionMemberships(ctx context.Context, regionID string) ([]regions.OrgRegionMembership, error)
	BindOrgToRegion(ctx context.Context, regionID, orgID, status string) (*regions.OrgRegionMembership, error)
	BindAllOrgsToRegion(ctx context.Context, regionID string) (int, error)
	UnbindOrgFromRegion(ctx context.Context, regionID, orgID string) error
	GetRegionOverlay(ctx context.Context, regionID, orgID string) (*regions.RegionConfigOverlay, error)
	PutRegionOverlay(ctx context.Context, regionID, orgID string, payload regions.OverlayPayload) (*regions.RegionConfigOverlay, error)
	DeleteRegionOverlay(ctx context.Context, regionID, orgID string) error

	ListFederationPeers(ctx context.Context) ([]federation.ControlPlanePeer, error)
	GetFederationPeer(ctx context.Context, peerID string) (*federation.ControlPlanePeer, error)
	RegisterFederationPeer(ctx context.Context, name, regionID, baseURL string) (*federation.PeerWithToken, error)
	UpdateFederationPeer(ctx context.Context, peerID, name, baseURL, status string) (*federation.ControlPlanePeer, error)
	RotateFederationPeerJoinToken(ctx context.Context, peerID string) (*federation.PeerWithToken, error)
	AuthenticateFederationPeerToken(ctx context.Context, rawToken string) (*federation.ControlPlanePeer, error)
	JoinFederationPeer(ctx context.Context, rawToken string) (*federation.ControlPlanePeer, error)
	ExportFederationRegion(ctx context.Context, slug string, since int64, objectPrefix string) (*federation.RegionExport, error)
	RecordFederationPeerSync(ctx context.Context, peerID string, cursor int64, syncErr string) error
}

// Service orchestrates platform queries and commands (mutate + publish + events).
type Service struct {
	API    ConfigAPI
	Snap   SnapshotPublisher
	Events EventRecorder
}

func New(api ConfigAPI, snap SnapshotPublisher) *Service {
	return &Service{API: api, Snap: snap, Events: NopRecorder{}}
}
