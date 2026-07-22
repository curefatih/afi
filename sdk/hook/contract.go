// Package hook defines the stable in-process lifecycle contract for gateway extensions.
//
// BeforeCall runs after auth on every modality and may mutate CallContext or deny.
// AfterCall runs after the upstream attempt finishes.
// BeforeChat / AfterChat remain available for chat body mutation and logging.
package hook

import (
	"context"
)

// Principal is the authenticated API key identity for a call.
type Principal struct {
	OrganizationID string
	ProjectID      string
	APIKeyID       string
	Kind           string
	OwnerUserID    string
	Name           string
}

// RouteContext describes the requested operation (before or after provider binding).
type RouteContext struct {
	Model    string
	Path     string
	Stream   bool
	Modality string
}

// CallContext is the mutable request-scoped bag passed through BeforeCall / AfterCall.
type CallContext struct {
	Principal Principal
	Route     RouteContext
	Tags      map[string]string
	// Headers are sanitized inbound HTTP headers (lowercased keys) for policy/CEL.
	Headers  map[string]string
	Metadata map[string]any
	Body     []byte
	// RequestHeaders are applied to the upstream provider HTTP request after BeforeCall.
	// Keys are canonicalized with http.CanonicalHeaderKey when applied.
	RequestHeaders map[string]string
	// ResponseHeaders are merged onto the client HTTP response (allow and deny paths).
	ResponseHeaders map[string]string
}

// CallDecision is returned by BeforeCall. Allow=false stops the request.
type CallDecision struct {
	Allow   bool
	Status  int               // default 403 when deny and unset
	Reason  string            // machine code, e.g. policy_violation
	Message string            // human-readable; optional
	Headers map[string]string // optional response headers on deny
}

// Allow returns an allowing decision.
func Allow() CallDecision {
	return CallDecision{Allow: true}
}

// Deny returns a denying decision with the given HTTP status and reason code.
func Deny(status int, reason, message string) CallDecision {
	if status == 0 {
		status = 403
	}
	return CallDecision{Allow: false, Status: status, Reason: reason, Message: message}
}

// BeforeCallHook runs after authentication and may mutate call or deny.
type BeforeCallHook interface {
	Name() string
	BeforeCall(ctx context.Context, call *CallContext) (CallDecision, error)
}

// AfterCallInfo is passed to AfterCallHook after an upstream attempt finishes.
type AfterCallInfo struct {
	Status           string
	LatencyMs        int64
	ProviderType     string
	TargetModel      string
	PromptTokens     int64
	CompletionTokens int64
}

// AfterCallHook runs after the attempt completes (success or error).
type AfterCallHook interface {
	Name() string
	AfterCall(ctx context.Context, call *CallContext, info AfterCallInfo) error
}

// ChatHook mutates the OpenAI chat request body before provider dispatch.
type ChatHook interface {
	Name() string
	BeforeChat(ctx context.Context, body []byte) ([]byte, error)
}

// AfterChatInfo is passed to AfterChatHook after a chat attempt finishes.
type AfterChatInfo struct {
	Model        string
	Status       string
	LatencyMs    int64
	ProviderType string
	TargetModel  string
}

// AfterChatHook runs after the chat attempt completes (success or error).
type AfterChatHook interface {
	Name() string
	AfterChat(ctx context.Context, info AfterChatInfo) error
}
