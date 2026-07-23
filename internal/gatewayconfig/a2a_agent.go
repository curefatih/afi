package gatewayconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

// A2AAgent is the write-model for an org-scoped A2A upstream agent.
type A2AAgent struct {
	ID             string          `json:"id"`
	OrganizationID string          `json:"organization_id"`
	Alias          string          `json:"alias"`
	Name           string          `json:"name"`
	UpstreamURL    string          `json:"upstream_url"`
	CardURL        string          `json:"card_url"`
	CardCache      json.RawMessage `json:"card_cache,omitempty"`
	APIKeyEnv      string          `json:"api_key_env"`
	AuthScheme     string          `json:"auth_scheme"`
	Enabled        bool            `json:"enabled"`
	CreatedAt      time.Time       `json:"created_at"`
}

// A2AAgentRepository persists write-model A2A agents.
type A2AAgentRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]A2AAgent, error)
	Insert(ctx context.Context, a A2AAgent) error
	Get(ctx context.Context, id string) (*A2AAgent, error)
	Update(ctx context.Context, a A2AAgent) (*A2AAgent, error)
	Delete(ctx context.Context, id string) error
	OrgID(ctx context.Context, id string) (string, error)
}

// ParseA2AAlias validates a URL path–safe alias slug (same rules as MCP).
func ParseA2AAlias(alias string) (string, error) {
	return ParseMCPAlias(alias)
}

// ParseA2AURL validates an absolute http(s) URL.
func ParseA2AURL(raw string) (string, error) {
	return ParseMCPBaseURL(raw)
}

func normalizeCardCache(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("%w: card_cache must be valid JSON", kernel.ErrInvalidRequest)
	}
	return raw, nil
}

// NewA2AAgent builds a validated entity.
func NewA2AAgent(
	id, orgID, alias, name, upstreamURL, cardURL, apiKeyEnv, authScheme string,
	cardCache json.RawMessage,
	enabled bool,
	now time.Time,
) (*A2AAgent, error) {
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
	}
	a, err := ParseA2AAlias(alias)
	if err != nil {
		return nil, err
	}
	upstream, err := ParseA2AURL(upstreamURL)
	if err != nil {
		return nil, err
	}
	cardURL = strings.TrimSpace(cardURL)
	if cardURL != "" {
		parsed, err := ParseA2AURL(cardURL)
		if err != nil {
			return nil, fmt.Errorf("%w: card_url must be an absolute http(s) URL", kernel.ErrInvalidRequest)
		}
		cardURL = parsed
	}
	cache, err := normalizeCardCache(cardCache)
	if err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &A2AAgent{
		ID:             id,
		OrganizationID: orgID,
		Alias:          a,
		Name:           name,
		UpstreamURL:    upstream,
		CardURL:        cardURL,
		CardCache:      cache,
		APIKeyEnv:      strings.TrimSpace(apiKeyEnv),
		AuthScheme:     strings.TrimSpace(authScheme),
		Enabled:        enabled,
		CreatedAt:      now.UTC(),
	}, nil
}

// ApplyA2AAgentPatch mutates optional fields.
func ApplyA2AAgentPatch(
	cur *A2AAgent,
	alias, name, upstreamURL, cardURL, apiKeyEnv, authScheme *string,
	cardCache json.RawMessage,
	enabled *bool,
) error {
	if cur == nil {
		return kernel.ErrNotFound
	}
	if alias != nil {
		a, err := ParseA2AAlias(*alias)
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
	if upstreamURL != nil {
		u, err := ParseA2AURL(*upstreamURL)
		if err != nil {
			return err
		}
		cur.UpstreamURL = u
	}
	if cardURL != nil {
		c := strings.TrimSpace(*cardURL)
		if c == "" {
			cur.CardURL = ""
		} else {
			u, err := ParseA2AURL(c)
			if err != nil {
				return fmt.Errorf("%w: card_url invalid", kernel.ErrInvalidRequest)
			}
			cur.CardURL = u
		}
	}
	if apiKeyEnv != nil {
		cur.APIKeyEnv = strings.TrimSpace(*apiKeyEnv)
	}
	if authScheme != nil {
		cur.AuthScheme = strings.TrimSpace(*authScheme)
	}
	if cardCache != nil {
		cache, err := normalizeCardCache(cardCache)
		if err != nil {
			return err
		}
		cur.CardCache = cache
	}
	if enabled != nil {
		cur.Enabled = *enabled
	}
	return nil
}

// CreateA2AAgent validates and persists.
func CreateA2AAgent(
	ctx context.Context,
	repo A2AAgentRepository,
	id, orgID, alias, name, upstreamURL, cardURL, apiKeyEnv, authScheme string,
	cardCache json.RawMessage,
	enabled bool,
) (*A2AAgent, error) {
	a, err := NewA2AAgent(id, orgID, alias, name, upstreamURL, cardURL, apiKeyEnv, authScheme, cardCache, enabled, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *a); err != nil {
		return nil, err
	}
	return a, nil
}

// UpdateA2AAgent loads, patches, and persists.
func UpdateA2AAgent(
	ctx context.Context,
	repo A2AAgentRepository,
	id string,
	alias, name, upstreamURL, cardURL, apiKeyEnv, authScheme *string,
	cardCache json.RawMessage,
	enabled *bool,
) (*A2AAgent, error) {
	cur, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ApplyA2AAgentPatch(cur, alias, name, upstreamURL, cardURL, apiKeyEnv, authScheme, cardCache, enabled); err != nil {
		return nil, err
	}
	return repo.Update(ctx, *cur)
}
