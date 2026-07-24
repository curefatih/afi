package controlplane

import (
	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/credentials"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/curefatih/afi/internal/usage"
)

const (
	OrgRoleOwner  = tenancy.OrgRoleOwner
	OrgRoleAdmin  = tenancy.OrgRoleAdmin
	OrgRoleMember = tenancy.OrgRoleMember

	TeamRoleOwner  = tenancy.TeamRoleOwner
	TeamRoleAdmin  = tenancy.TeamRoleAdmin
	TeamRoleMember = tenancy.TeamRoleMember

	InviteStatusPending = tenancy.InviteStatusPending
)

type User = identity.User
type Organization = tenancy.Organization
type Team = tenancy.Team
type TeamMember = tenancy.TeamMember
type Project = tenancy.Project
type Environment = tenancy.Environment
type OrgMember = tenancy.OrgMember
type OrgInvite = tenancy.OrgInvite
type InviteOutcome = tenancy.InviteOutcome
type InvitePreview = tenancy.InvitePreview

type APIKey = access.APIKey

type Provider = gatewayconfig.Provider
type RouteFallback = gatewayconfig.RouteFallback
type RetryConfig = gatewayconfig.RetryConfig
type ObjectStoreConfig = gatewayconfig.ObjectStoreConfig
type Route = gatewayconfig.Route
type Quota = gatewayconfig.Quota
type RequestPolicy = gatewayconfig.RequestPolicy
type WasmHook = gatewayconfig.WasmHook
type MCPBackend = gatewayconfig.MCPBackend
type A2AAgent = gatewayconfig.A2AAgent

type Credential = credentials.Credential
type CredentialAssignment = credentials.Assignment

type UsageEvent = usage.Record
type UsageFilter = usage.Filter
type UsageSummaryBucket = usage.SummaryBucket
type ModelPrice = usage.ModelPrice
type ProviderHealth = usage.ProviderHealth
