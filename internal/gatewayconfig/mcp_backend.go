package gatewayconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

var mcpAliasPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// MCPBackend is the write-model for an org-scoped MCP Streamable HTTP upstream.
type MCPBackend struct {
	ID              string          `json:"id"`
	OrganizationID  string          `json:"organization_id"`
	Alias           string          `json:"alias"`
	Name            string          `json:"name"`
	BaseURL         string          `json:"base_url"`
	APIKeyEnv       string          `json:"api_key_env"`
	MethodAllowlist json.RawMessage `json:"method_allowlist"`
	Enabled         bool            `json:"enabled"`
	CreatedAt       time.Time       `json:"created_at"`
}

// MCPBackendRepository persists write-model MCP backends.
type MCPBackendRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]MCPBackend, error)
	Insert(ctx context.Context, b MCPBackend) error
	Get(ctx context.Context, id string) (*MCPBackend, error)
	Update(ctx context.Context, b MCPBackend) (*MCPBackend, error)
	Delete(ctx context.Context, id string) error
	OrgID(ctx context.Context, id string) (string, error)
}

// ParseMCPAlias validates a URL path–safe alias slug.
func ParseMCPAlias(alias string) (string, error) {
	alias = strings.ToLower(strings.TrimSpace(alias))
	if alias == "" {
		return "", fmt.Errorf("%w: alias required", kernel.ErrInvalidRequest)
	}
	if !mcpAliasPattern.MatchString(alias) {
		return "", fmt.Errorf("%w: alias must be lowercase alphanumeric with optional hyphens (1-63 chars)", kernel.ErrInvalidRequest)
	}
	return alias, nil
}

// ParseMCPBaseURL validates an absolute http(s) URL.
func ParseMCPBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%w: base_url required", kernel.ErrInvalidRequest)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("%w: base_url must be an absolute http(s) URL", kernel.ErrInvalidRequest)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%w: base_url scheme must be http or https", kernel.ErrInvalidRequest)
	}
	return strings.TrimRight(raw, "/"), nil
}

// NormalizeMethodAllowlist validates a JSON array of method strings.
func NormalizeMethodAllowlist(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`[]`), nil
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("%w: method_allowlist must be valid JSON", kernel.ErrInvalidRequest)
	}
	var methods []string
	if err := json.Unmarshal(raw, &methods); err != nil {
		return nil, fmt.Errorf("%w: method_allowlist must be a JSON array of strings", kernel.ErrInvalidRequest)
	}
	out := make([]string, 0, len(methods))
	for _, m := range methods {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		out = append(out, m)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// NewMCPBackend builds a validated entity.
func NewMCPBackend(
	id, orgID, alias, name, baseURL, apiKeyEnv string,
	methodAllowlist json.RawMessage,
	enabled bool,
	now time.Time,
) (*MCPBackend, error) {
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
	}
	a, err := ParseMCPAlias(alias)
	if err != nil {
		return nil, err
	}
	base, err := ParseMCPBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	allow, err := NormalizeMethodAllowlist(methodAllowlist)
	if err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &MCPBackend{
		ID:              id,
		OrganizationID:  orgID,
		Alias:           a,
		Name:            name,
		BaseURL:         base,
		APIKeyEnv:       strings.TrimSpace(apiKeyEnv),
		MethodAllowlist: allow,
		Enabled:         enabled,
		CreatedAt:       now.UTC(),
	}, nil
}

// ApplyMCPBackendPatch mutates optional fields.
func ApplyMCPBackendPatch(
	cur *MCPBackend,
	alias, name, baseURL, apiKeyEnv *string,
	methodAllowlist json.RawMessage,
	enabled *bool,
) error {
	if cur == nil {
		return kernel.ErrNotFound
	}
	if alias != nil {
		a, err := ParseMCPAlias(*alias)
		if err != nil {
			return err
		}
		cur.Alias = a
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
		}
		cur.Name = n
	}
	if baseURL != nil {
		base, err := ParseMCPBaseURL(*baseURL)
		if err != nil {
			return err
		}
		cur.BaseURL = base
	}
	if apiKeyEnv != nil {
		cur.APIKeyEnv = strings.TrimSpace(*apiKeyEnv)
	}
	if methodAllowlist != nil {
		allow, err := NormalizeMethodAllowlist(methodAllowlist)
		if err != nil {
			return err
		}
		cur.MethodAllowlist = allow
	}
	if enabled != nil {
		cur.Enabled = *enabled
	}
	return nil
}

// CreateMCPBackend validates and persists.
func CreateMCPBackend(
	ctx context.Context,
	repo MCPBackendRepository,
	id, orgID, alias, name, baseURL, apiKeyEnv string,
	methodAllowlist json.RawMessage,
	enabled bool,
) (*MCPBackend, error) {
	b, err := NewMCPBackend(id, orgID, alias, name, baseURL, apiKeyEnv, methodAllowlist, enabled, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *b); err != nil {
		return nil, err
	}
	return b, nil
}

// UpdateMCPBackend loads, patches, and persists.
func UpdateMCPBackend(
	ctx context.Context,
	repo MCPBackendRepository,
	id string,
	alias, name, baseURL, apiKeyEnv *string,
	methodAllowlist json.RawMessage,
	enabled *bool,
) (*MCPBackend, error) {
	cur, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ApplyMCPBackendPatch(cur, alias, name, baseURL, apiKeyEnv, methodAllowlist, enabled); err != nil {
		return nil, err
	}
	return repo.Update(ctx, *cur)
}
