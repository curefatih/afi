package regions

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

const (
	MembershipStatusActive   = "active"
	MembershipStatusDisabled = "disabled"
)

// OrgRegionMembership binds an organization to a region.
type OrgRegionMembership struct {
	OrganizationID string    `json:"organization_id"`
	RegionID       string    `json:"region_id"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RegionConfigOverlay is a full replacement gateway config slice for one org in one region.
// When present, BuildRegionSource uses this instead of the org's base config.
type RegionConfigOverlay struct {
	OrganizationID string    `json:"organization_id"`
	RegionID       string    `json:"region_id"`
	Payload        OverlayPayload `json:"payload"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// OverlayPayload holds gateway-relevant config kinds (snapshot compile shapes).
type OverlayPayload struct {
	Providers      []snapshot.Provider             `json:"providers,omitempty"`
	Routes         []snapshot.Route                `json:"routes,omitempty"`
	Quotas         []snapshot.Quota                `json:"quotas,omitempty"`
	Policies       []snapshot.Policy               `json:"policies,omitempty"`
	WasmHooks      []snapshot.WasmHook             `json:"wasm_hooks,omitempty"`
	MCPBackends    []snapshot.MCPBackend           `json:"mcp_backends,omitempty"`
	A2AAgents      []snapshot.A2AAgent             `json:"a2a_agents,omitempty"`
	Credentials    []snapshot.Credential           `json:"credentials,omitempty"`
	Assignments    []snapshot.CredentialAssignment `json:"assignments,omitempty"`
	DefaultRetry   *snapshot.RetryConfig           `json:"default_retry,omitempty"`
	ObjectStore    *snapshot.ObjectStoreConfig     `json:"object_store,omitempty"`
}

// ValidateMembershipStatus checks status is active or disabled.
func ValidateMembershipStatus(status string) error {
	switch status {
	case MembershipStatusActive, MembershipStatusDisabled:
		return nil
	default:
		return fmt.Errorf("%w: status must be active or disabled", kernel.ErrInvalidRequest)
	}
}

// NewOrgRegionMembership builds a validated membership.
func NewOrgRegionMembership(orgID, regionID, status string, now time.Time) (*OrgRegionMembership, error) {
	orgID = strings.TrimSpace(orgID)
	regionID = strings.TrimSpace(regionID)
	status = strings.TrimSpace(status)
	if orgID == "" || regionID == "" {
		return nil, fmt.Errorf("%w: organization_id and region_id required", kernel.ErrInvalidRequest)
	}
	if status == "" {
		status = MembershipStatusActive
	}
	if err := ValidateMembershipStatus(status); err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	return &OrgRegionMembership{
		OrganizationID: orgID, RegionID: regionID, Status: status,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// NewRegionConfigOverlay builds an overlay entity; payload must be valid JSON-serializable.
func NewRegionConfigOverlay(orgID, regionID string, payload OverlayPayload, now time.Time) (*RegionConfigOverlay, error) {
	orgID = strings.TrimSpace(orgID)
	regionID = strings.TrimSpace(regionID)
	if orgID == "" || regionID == "" {
		return nil, fmt.Errorf("%w: organization_id and region_id required", kernel.ErrInvalidRequest)
	}
	// Ensure org IDs on nested rows match.
	for i := range payload.Routes {
		payload.Routes[i].OrganizationID = orgID
	}
	for i := range payload.Quotas {
		payload.Quotas[i].OrganizationID = orgID
	}
	for i := range payload.Policies {
		payload.Policies[i].OrganizationID = orgID
	}
	for i := range payload.WasmHooks {
		payload.WasmHooks[i].OrganizationID = orgID
	}
	for i := range payload.MCPBackends {
		payload.MCPBackends[i].OrganizationID = orgID
	}
	for i := range payload.A2AAgents {
		payload.A2AAgents[i].OrganizationID = orgID
	}
	for i := range payload.Credentials {
		payload.Credentials[i].OrganizationID = orgID
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	return &RegionConfigOverlay{
		OrganizationID: orgID, RegionID: regionID, Payload: payload,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// MarshalPayload encodes overlay payload as JSON.
func (p OverlayPayload) MarshalPayload() ([]byte, error) {
	return json.Marshal(p)
}

// UnmarshalOverlayPayload decodes JSON into OverlayPayload.
func UnmarshalOverlayPayload(raw []byte) (OverlayPayload, error) {
	var p OverlayPayload
	if len(raw) == 0 {
		return p, nil
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, fmt.Errorf("%w: invalid overlay payload: %v", kernel.ErrInvalidRequest, err)
	}
	return p, nil
}
