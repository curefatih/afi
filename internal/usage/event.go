package usage

// Event is the canonical usage record emitted by the gateway and drained by workers.
// Control-plane list/summary responses may enrich this with IDs, cost, and join fields.
type Event struct {
	OrganizationID   string         `json:"organization_id"`
	ProjectID        string         `json:"project_id"`
	APIKeyID         string         `json:"api_key_id"`
	Model            string         `json:"model"`
	ProviderType     string         `json:"provider_type"`
	TargetModel      string         `json:"target_model"`
	Status           string         `json:"status"`
	LatencyMs        int64          `json:"latency_ms"`
	PromptTokens     int64          `json:"prompt_tokens"`
	CompletionTokens int64          `json:"completion_tokens"`
	Modality         string         `json:"modality"`
	Metrics          map[string]any `json:"metrics,omitempty"`
}
