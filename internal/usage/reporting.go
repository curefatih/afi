package usage

import "time"

// Record is a persisted usage row enriched for control-plane list APIs.
type Record struct {
	ID               int64             `json:"id"`
	OrganizationID   string            `json:"organization_id"`
	ProjectID        string            `json:"project_id"`
	ProjectName      string            `json:"project_name,omitempty"`
	APIKeyID         string            `json:"api_key_id"`
	CredentialID     string            `json:"credential_id,omitempty"`
	CredentialName   string            `json:"credential_name,omitempty"`
	UsedBYOK         bool              `json:"used_byok"`
	Model            string            `json:"model"`
	ProviderType     string            `json:"provider_type,omitempty"`
	Status           string            `json:"status"`
	LatencyMs        int64             `json:"latency_ms"`
	PromptTokens     int64             `json:"prompt_tokens"`
	CompletionTokens int64             `json:"completion_tokens"`
	Modality         string            `json:"modality"`
	Metrics          map[string]any    `json:"metrics"`
	Tags             map[string]string `json:"tags,omitempty"`
	CostUSD          *float64          `json:"cost_usd,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	KeyName          string            `json:"key_name,omitempty"`
	KeyKind          string            `json:"key_kind,omitempty"`
	OwnerUserID      string            `json:"owner_user_id,omitempty"`
	OwnerEmail       string            `json:"owner_email,omitempty"`
	OwnerName        string            `json:"owner_name,omitempty"`
}

// Filter constrains usage list and summary queries.
type Filter struct {
	Limit        int
	ProjectID    string
	APIKeyID     string
	CredentialID string
	Model        string
	Modality     string
	ExcludeBYOK  bool // when true, only rows where used_byok=false
	BYOKOnly     bool // when true, only rows where used_byok=true
	From         *time.Time
	To           *time.Time
	GroupBy      string // day | model | key | modality | byok (summary only)
}

// SummaryBucket is one row of a usage summary aggregation.
type SummaryBucket struct {
	Bucket           string             `json:"bucket"`
	Label            string             `json:"label"`
	Requests         int64              `json:"requests"`
	CostUSD          float64            `json:"cost_usd"`
	PromptTokens     int64              `json:"prompt_tokens"`
	CompletionTokens int64              `json:"completion_tokens"`
	MetricsTotals    map[string]float64 `json:"metrics_totals,omitempty"`
	KeyKind          string             `json:"key_kind,omitempty"`
	OwnerEmail       string             `json:"owner_email,omitempty"`
	OwnerName        string             `json:"owner_name,omitempty"`
}

// ModelPrice is a pricing row used for cost estimation.
type ModelPrice struct {
	ProviderType  string
	Model         string
	InputPerMTok  float64
	OutputPerMTok float64
}

// ProviderHealth aggregates request outcomes for a provider.
type ProviderHealth struct {
	ProviderID   string  `json:"provider_id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Requests     int64   `json:"requests"`
	Errors       int64   `json:"errors"`
	ErrorRate    float64 `json:"error_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	Status       string  `json:"status"` // healthy | degraded | down | unknown
}

// ClassifyProviderHealth maps request/error stats to a status label.
func ClassifyProviderHealth(requests, errors int64, errorRate float64) string {
	if requests == 0 {
		return "unknown"
	}
	if errorRate >= 0.9 || (requests >= 3 && errors == requests) {
		return "down"
	}
	if errorRate >= 0.2 {
		return "degraded"
	}
	return "healthy"
}
