package gatewayconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

// Wasm hook phases compiled into the gateway snapshot.
const (
	WasmPhaseBeforeCall = "before_call"
	WasmPhaseBeforeChat = "before_chat"
	WasmPhaseAfterCall  = "after_call"
)

// WasmHook is the write-model for an org-scoped sandboxed lifecycle hook.
type WasmHook struct {
	ID             string          `json:"id"`
	OrganizationID string          `json:"organization_id"`
	Name           string          `json:"name"`
	Phase          string          `json:"phase"`
	ModuleURI      string          `json:"module_uri"`
	Digest         string          `json:"digest"`
	Enabled        bool            `json:"enabled"`
	Priority       int             `json:"priority"`
	Config         json.RawMessage `json:"config"`
	CreatedAt      time.Time       `json:"created_at"`
}

// WasmHookRepository persists write-model WASM hooks.
type WasmHookRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]WasmHook, error)
	Insert(ctx context.Context, h WasmHook) error
	Get(ctx context.Context, id string) (*WasmHook, error)
	Update(ctx context.Context, h WasmHook) (*WasmHook, error)
	Delete(ctx context.Context, id string) error
	OrgID(ctx context.Context, id string) (string, error)
}

// ParseWasmPhase validates a lifecycle phase.
func ParseWasmPhase(phase string) (string, error) {
	switch strings.TrimSpace(phase) {
	case WasmPhaseBeforeCall, WasmPhaseBeforeChat, WasmPhaseAfterCall:
		return strings.TrimSpace(phase), nil
	default:
		return "", fmt.Errorf("%w: phase must be before_call, before_chat, or after_call", kernel.ErrInvalidRequest)
	}
}

// NewWasmHook builds a validated entity.
func NewWasmHook(id, orgID, name, phase, moduleURI, digest string, enabled bool, priority int, config json.RawMessage, now time.Time) (*WasmHook, error) {
	name = strings.TrimSpace(name)
	moduleURI = strings.TrimSpace(moduleURI)
	digest = strings.TrimSpace(strings.ToLower(digest))
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
	}
	ph, err := ParseWasmPhase(phase)
	if err != nil {
		return nil, err
	}
	if moduleURI == "" {
		return nil, fmt.Errorf("%w: module_uri required", kernel.ErrInvalidRequest)
	}
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	} else if !json.Valid(config) {
		return nil, fmt.Errorf("%w: config must be valid JSON", kernel.ErrInvalidRequest)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &WasmHook{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Phase:          ph,
		ModuleURI:      moduleURI,
		Digest:         digest,
		Enabled:        enabled,
		Priority:       priority,
		Config:         config,
		CreatedAt:      now.UTC(),
	}, nil
}

// ApplyWasmHookPatch mutates optional fields.
func ApplyWasmHookPatch(cur *WasmHook, name, phase, moduleURI, digest *string, enabled *bool, priority *int, config json.RawMessage) error {
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
	if phase != nil {
		ph, err := ParseWasmPhase(*phase)
		if err != nil {
			return err
		}
		cur.Phase = ph
	}
	if moduleURI != nil {
		u := strings.TrimSpace(*moduleURI)
		if u == "" {
			return fmt.Errorf("%w: module_uri required", kernel.ErrInvalidRequest)
		}
		cur.ModuleURI = u
	}
	if digest != nil {
		cur.Digest = strings.TrimSpace(strings.ToLower(*digest))
	}
	if enabled != nil {
		cur.Enabled = *enabled
	}
	if priority != nil {
		cur.Priority = *priority
	}
	if config != nil {
		if len(config) == 0 {
			cur.Config = json.RawMessage(`{}`)
		} else if !json.Valid(config) {
			return fmt.Errorf("%w: config must be valid JSON", kernel.ErrInvalidRequest)
		} else {
			cur.Config = config
		}
	}
	return nil
}

// CreateWasmHook validates and persists.
func CreateWasmHook(
	ctx context.Context,
	repo WasmHookRepository,
	id, orgID, name, phase, moduleURI, digest string,
	enabled bool,
	priority int,
	config json.RawMessage,
) (*WasmHook, error) {
	h, err := NewWasmHook(id, orgID, name, phase, moduleURI, digest, enabled, priority, config, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *h); err != nil {
		return nil, err
	}
	return h, nil
}

// UpdateWasmHook loads, patches, and persists.
func UpdateWasmHook(
	ctx context.Context,
	repo WasmHookRepository,
	id string,
	name, phase, moduleURI, digest *string,
	enabled *bool,
	priority *int,
	config json.RawMessage,
) (*WasmHook, error) {
	cur, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ApplyWasmHookPatch(cur, name, phase, moduleURI, digest, enabled, priority, config); err != nil {
		return nil, err
	}
	return repo.Update(ctx, *cur)
}
