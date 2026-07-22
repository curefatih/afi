package usage

// Event is the canonical usage record emitted by the gateway and drained by workers.
// Control-plane list/summary responses may enrich this with IDs, cost, and join fields.
type Event struct {
	OrganizationID   string            `json:"organization_id"`
	ProjectID        string            `json:"project_id"`
	APIKeyID         string            `json:"api_key_id"`
	CredentialID     string            `json:"credential_id,omitempty"`
	UsedBYOK         bool              `json:"used_byok"`
	Model            string            `json:"model"`
	ProviderType     string            `json:"provider_type"`
	TargetModel      string            `json:"target_model"`
	Status           string            `json:"status"`
	LatencyMs        int64             `json:"latency_ms"`
	PromptTokens     int64             `json:"prompt_tokens"`
	CompletionTokens int64             `json:"completion_tokens"`
	Modality         string            `json:"modality"`
	Metrics          map[string]any    `json:"metrics,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// MarkBYOK sets UsedBYOK from whether a stored credential was used (vs platform api_key_env).
func (e *Event) MarkBYOK() {
	e.UsedBYOK = e.CredentialID != ""
}
