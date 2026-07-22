package gatewayconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

// Policy action kinds (THEN clause).
const (
	ActionAllow         = "allow"
	ActionDeny          = "deny"
	ActionSetHeader     = "set_header"
	ActionUseCredential = "use_credential"
)

// PolicyAction is one Then step for a when/then policy.
type PolicyAction struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config,omitempty"`
}

// RequestPolicy is a when/then CEL policy for an organization.
// WHEN Expression is true, THEN Actions run in order.
type RequestPolicy struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Expression     string         `json:"expression"`
	Actions        []PolicyAction `json:"actions"`
	Enabled        bool           `json:"enabled"`
	Priority       int            `json:"priority"`
	CreatedAt      time.Time      `json:"created_at"`
}

// ActionConfigAllow/Deny have no fields.
type ActionConfigSetHeader struct {
	Header    string `json:"header"`
	Value     string `json:"value,omitempty"`
	ValueExpr string `json:"value_expr,omitempty"` // CEL → string; wins over Value when set
}

type ActionConfigUseCredential struct {
	CredentialName     string `json:"credential_name,omitempty"`
	CredentialNameExpr string `json:"credential_name_expr,omitempty"` // CEL → string; wins when set
}

// ExpressionValidator checks CEL expressions without owning the engine.
type ExpressionValidator interface {
	Validate(expression string) error
	ValidateString(expression string) error
}

// PolicyRepository persists write-model request policies.
type PolicyRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]RequestPolicy, error)
	Insert(ctx context.Context, p RequestPolicy) error
	Get(ctx context.Context, policyID string) (*RequestPolicy, error)
	Update(ctx context.Context, p RequestPolicy) (*RequestPolicy, error)
	UpdatePriorities(ctx context.Context, orgID string, items []PolicyPriorityUpdate) error
	Delete(ctx context.Context, policyID string) error
	OrgID(ctx context.Context, policyID string) (string, error)
}

// PolicyPriorityUpdate is one policy's new priority in a batch reorder.
type PolicyPriorityUpdate struct {
	ID       string `json:"id"`
	Priority int    `json:"priority"`
}

// NormalizeAction validates one Then action and returns a canonical copy.
func NormalizeAction(a PolicyAction, v ExpressionValidator) (PolicyAction, error) {
	typ, cfg, err := NormalizeActionConfig(a.Type, a.Config, v)
	if err != nil {
		return PolicyAction{}, err
	}
	return PolicyAction{Type: typ, Config: cfg}, nil
}

// NormalizeActions validates a Then list (at least one action required).
func NormalizeActions(actions []PolicyAction, v ExpressionValidator) ([]PolicyAction, error) {
	if len(actions) == 0 {
		return nil, fmt.Errorf("%w: at least one then action required", kernel.ErrInvalidRequest)
	}
	out := make([]PolicyAction, 0, len(actions))
	for i, a := range actions {
		na, err := NormalizeAction(a, v)
		if err != nil {
			return nil, fmt.Errorf("actions[%d]: %w", i, err)
		}
		out = append(out, na)
	}
	return out, nil
}

// NormalizeActionConfig validates action + config and returns canonical JSON.
// When v is non-nil, dynamic *_expr fields are validated as string CEL.
func NormalizeActionConfig(action string, raw json.RawMessage, v ExpressionValidator) (string, json.RawMessage, error) {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = ActionDeny
	}
	switch action {
	case ActionAllow, ActionDeny:
		return action, json.RawMessage(`{}`), nil
	case ActionSetHeader:
		var cfg ActionConfigSetHeader
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return "", nil, fmt.Errorf("%w: invalid set_header config: %v", kernel.ErrInvalidRequest, err)
			}
		}
		cfg.Header = strings.TrimSpace(cfg.Header)
		cfg.Value = strings.TrimSpace(cfg.Value)
		cfg.ValueExpr = strings.TrimSpace(cfg.ValueExpr)
		if cfg.Header == "" {
			return "", nil, fmt.Errorf("%w: set_header requires header", kernel.ErrInvalidRequest)
		}
		if cfg.ValueExpr != "" {
			if v != nil {
				if err := v.ValidateString(cfg.ValueExpr); err != nil {
					return "", nil, fmt.Errorf("%w: invalid value_expr: %v", kernel.ErrInvalidRequest, err)
				}
			}
			cfg.Value = ""
		}
		out, err := json.Marshal(cfg)
		if err != nil {
			return "", nil, err
		}
		return action, out, nil
	case ActionUseCredential:
		var cfg ActionConfigUseCredential
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return "", nil, fmt.Errorf("%w: invalid use_credential config: %v", kernel.ErrInvalidRequest, err)
			}
		}
		cfg.CredentialName = strings.TrimSpace(cfg.CredentialName)
		cfg.CredentialNameExpr = strings.TrimSpace(cfg.CredentialNameExpr)
		if cfg.CredentialNameExpr != "" {
			if v != nil {
				if err := v.ValidateString(cfg.CredentialNameExpr); err != nil {
					return "", nil, fmt.Errorf("%w: invalid credential_name_expr: %v", kernel.ErrInvalidRequest, err)
				}
			}
			cfg.CredentialName = ""
		} else if cfg.CredentialName == "" {
			return "", nil, fmt.Errorf("%w: use_credential requires credential_name or credential_name_expr", kernel.ErrInvalidRequest)
		}
		out, err := json.Marshal(cfg)
		if err != nil {
			return "", nil, err
		}
		return action, out, nil
	default:
		return "", nil, fmt.Errorf("%w: unknown action %q (allow|deny|set_header|use_credential)", kernel.ErrInvalidRequest, action)
	}
}

// LegacySingleAction converts old action + action_config fields into an actions slice.
func LegacySingleAction(action string, actionConfig json.RawMessage) []PolicyAction {
	action = strings.TrimSpace(action)
	if action == "" {
		return nil
	}
	cfg := actionConfig
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	return []PolicyAction{{Type: action, Config: cfg}}
}

// NewRequestPolicy builds a validated policy entity.
func NewRequestPolicy(id, orgID, name, expression string, actions []PolicyAction, enabled bool, priority int, now time.Time, v ExpressionValidator) (*RequestPolicy, error) {
	name = strings.TrimSpace(name)
	expression = strings.TrimSpace(expression)
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	if name == "" || expression == "" {
		return nil, fmt.Errorf("%w: name and expression required", kernel.ErrInvalidRequest)
	}
	if v != nil {
		if err := v.Validate(expression); err != nil {
			return nil, fmt.Errorf("%w: invalid CEL expression: %v", kernel.ErrInvalidRequest, err)
		}
	}
	actions, err := NormalizeActions(actions, v)
	if err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &RequestPolicy{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Expression:     expression,
		Actions:        actions,
		Enabled:        enabled,
		Priority:       priority,
		CreatedAt:      now.UTC(),
	}, nil
}

// ApplyPolicyPatch mutates a policy with optional field updates.
func ApplyPolicyPatch(cur *RequestPolicy, name, expression *string, actions []PolicyAction, enabled *bool, priority *int, v ExpressionValidator) error {
	if cur == nil {
		return kernel.ErrNotFound
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
		}
		cur.Name = n
	}
	if expression != nil {
		e := strings.TrimSpace(*expression)
		if e == "" {
			return fmt.Errorf("%w: expression required", kernel.ErrInvalidRequest)
		}
		if v != nil {
			if err := v.Validate(e); err != nil {
				return fmt.Errorf("%w: invalid CEL expression: %v", kernel.ErrInvalidRequest, err)
			}
		}
		cur.Expression = e
	}
	if actions != nil {
		na, err := NormalizeActions(actions, v)
		if err != nil {
			return err
		}
		cur.Actions = na
	}
	if enabled != nil {
		cur.Enabled = *enabled
	}
	if priority != nil {
		cur.Priority = *priority
	}
	return nil
}
