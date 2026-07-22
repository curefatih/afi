package snapshot

// Policy action kinds (THEN clause) — keep in sync with gatewayconfig.
const (
	PolicyActionAllow         = "allow"
	PolicyActionDeny          = "deny"
	PolicyActionSetHeader     = "set_header"
	PolicyActionUseCredential = "use_credential"
)

// PolicyAction is one Then step compiled into the snapshot.
type PolicyAction struct {
	Type   string `json:"type"`
	Config []byte `json:"config,omitempty"`
}

// Policy is a when/then CEL rule compiled into the gateway snapshot.
// WHEN Expression is true, THEN Actions run in order.
type Policy struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Expression     string         `json:"expression"`
	Actions        []PolicyAction `json:"actions"`
	Enabled        bool           `json:"enabled"`
	Priority       int            `json:"priority"`
}

// EffectiveActions returns Then steps for evaluation.
func (p Policy) EffectiveActions() []PolicyAction {
	return p.Actions
}
