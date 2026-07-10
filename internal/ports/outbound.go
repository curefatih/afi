package ports

import (
	"context"

	"github.com/curefatih/afi/internal/core/domain"
)

// CredentialVault defines the secure retrieval mechanism for external provider API keys.
type CredentialVault interface {
	// GetProviderKey looks up the raw decrypted secret token for a provider given a platform project workspace context.
	GetProviderKey(ctx context.Context, projectID string, provider string) (string, error)
}

// LLMClient abstracts downstream foundational model connections (OpenAI, Anthropic, etc.).
type LLMClient interface {
	Call(ctx context.Context, req *domain.InternalRequest) (*domain.InternalResponse, error)
	StreamCall(ctx context.Context, req *domain.InternalRequest) (<-chan domain.StreamChunk, <-chan error)
}

// RouterService matches current state configurations against internal requests.
type RouterService interface {
	Route(req *domain.InternalRequest) (domain.TargetDestination, error)
	AddRule(ctx context.Context, rule domain.RoutingRule) error
}

// BudgetService calculates, monitors, and enforces tiered structural financial barriers.
type BudgetService interface {
	Check(ctx context.Context, ctxMeta domain.RequestContext) error
	CommitUsage(ctx context.Context, ctxMeta domain.RequestContext, usage domain.TokenUsage) error
}

// JSEngine provides a sandboxed execution runtime environment for untrusted user hooks.
type JSEngine interface {
	ExecuteHook(ctx context.Context, script string, stage domain.HookStage, payload any) (any, error)
}

// PluginService handles fast lookup retrieval of raw dynamic hook logic definitions.
type PluginService interface {
	GetHook(ctx context.Context, projectID string, stage domain.HookStage) (*domain.CustomPlugin, bool)
	SaveHook(ctx context.Context, projectID string, stage domain.HookStage, script string) error
}
