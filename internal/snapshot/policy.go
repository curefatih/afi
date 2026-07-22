package snapshot

// Policy action kinds (THEN clause) — keep in sync with gatewayconfig.
const (
	PolicyActionAllow         = "allow"
	PolicyActionDeny          = "deny"
	PolicyActionSetHeader     = "set_header"
	PolicyActionUseCredential = "use_credential"
)

// Policy is a when/then CEL rule compiled into the gateway snapshot.
// WHEN Expression is true, THEN Action runs using ActionConfig JSON.
type Policy struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Expression     string `json:"expression"`
	Action         string `json:"action"`
	ActionConfig   []byte `json:"action_config,omitempty"`
	Enabled        bool   `json:"enabled"`
	Priority       int    `json:"priority"`
}
