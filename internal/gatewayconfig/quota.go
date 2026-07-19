package gatewayconfig

import (
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

// Quota is the write-model quota configuration for an organization.
type Quota struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ScopeType      string    `json:"scope_type"`
	ScopeID        string    `json:"scope_id"`
	Metric         string    `json:"metric"`
	LimitValue     int64     `json:"limit_value"`
	Window         string    `json:"window"`
	CreatedAt      time.Time `json:"created_at"`
}

// NormalizeWindow defaults empty window to total.
func NormalizeWindow(window string) string {
	if window == "" {
		return snapshot.WindowTotal
	}
	return window
}

// ValidateWindow ensures the quota window is supported.
func ValidateWindow(window string) error {
	if !snapshot.ValidQuotaWindow(window) {
		return fmt.Errorf("%w: window must be total, minute, hour, or day", kernel.ErrInvalidRequest)
	}
	return nil
}

// ValidateMetric ensures the quota metric is supported.
func ValidateMetric(metric string) error {
	switch metric {
	case snapshot.MetricRequests, snapshot.MetricTokens:
		return nil
	case "":
		return fmt.Errorf("%w: metric is required", kernel.ErrInvalidRequest)
	default:
		return fmt.Errorf("%w: invalid metric %q", kernel.ErrInvalidRequest, metric)
	}
}

// ValidateScopeType ensures the scope type is known.
func ValidateScopeType(scopeType string) error {
	switch scopeType {
	case snapshot.ScopeOrganization, snapshot.ScopeProject, snapshot.ScopeUser, snapshot.ScopeAPIKey:
		return nil
	case "":
		return fmt.Errorf("%w: scope_type is required", kernel.ErrInvalidRequest)
	default:
		return fmt.Errorf("%w: invalid scope_type %q", kernel.ErrInvalidRequest, scopeType)
	}
}

// ValidateLimit ensures limit_value is positive.
func ValidateLimit(limitValue int64) error {
	if limitValue <= 0 {
		return fmt.Errorf("%w: limit_value must be positive", kernel.ErrInvalidRequest)
	}
	return nil
}

// ValidateNewFields checks pure field invariants before persistence.
func ValidateNewFields(orgID, scopeType, scopeID, metric string, limitValue int64, window string) error {
	if orgID == "" {
		return fmt.Errorf("%w: organization_id is required", kernel.ErrInvalidRequest)
	}
	if scopeID == "" {
		return fmt.Errorf("%w: scope_id is required", kernel.ErrInvalidRequest)
	}
	if err := ValidateScopeType(scopeType); err != nil {
		return err
	}
	if err := ValidateMetric(metric); err != nil {
		return err
	}
	if err := ValidateLimit(limitValue); err != nil {
		return err
	}
	if err := ValidateWindow(window); err != nil {
		return err
	}
	if scopeType == snapshot.ScopeOrganization && scopeID != orgID {
		return fmt.Errorf("%w: organization scope_id must match organization", kernel.ErrInvalidRequest)
	}
	return nil
}

// NewQuota builds a validated quota entity (does not check membership).
func NewQuota(id, orgID, scopeType, scopeID, metric string, limitValue int64, window string, now time.Time) (*Quota, error) {
	window = NormalizeWindow(window)
	scope, err := ParseQuotaScope(scopeType)
	if err != nil {
		return nil, err
	}
	met, err := ParseQuotaMetric(metric)
	if err != nil {
		return nil, err
	}
	win, err := ParseQuotaWindow(window)
	if err != nil {
		return nil, err
	}
	if err := ValidateNewFields(orgID, scope.String(), scopeID, met.String(), limitValue, win.String()); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", kernel.ErrInvalidRequest)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &Quota{
		ID:             id,
		OrganizationID: orgID,
		ScopeType:      scope.String(),
		ScopeID:        scopeID,
		Metric:         met.String(),
		LimitValue:     limitValue,
		Window:         win.String(),
		CreatedAt:      now.UTC(),
	}, nil
}
