package gatewayconfig

import "github.com/curefatih/afi/internal/snapshot"

// QuotaWindow is a typed quota time window.
type QuotaWindow string

const (
	WindowTotal  QuotaWindow = snapshot.WindowTotal
	WindowMinute QuotaWindow = snapshot.WindowMinute
	WindowHour   QuotaWindow = snapshot.WindowHour
	WindowDay    QuotaWindow = snapshot.WindowDay
)

// ParseQuotaWindow validates and returns a QuotaWindow. Empty defaults to total.
func ParseQuotaWindow(window string) (QuotaWindow, error) {
	window = NormalizeWindow(window)
	if err := ValidateWindow(window); err != nil {
		return "", err
	}
	return QuotaWindow(window), nil
}

func (w QuotaWindow) String() string { return string(w) }

// QuotaMetric is a typed quota metric.
type QuotaMetric string

const (
	MetricRequests QuotaMetric = snapshot.MetricRequests
	MetricTokens   QuotaMetric = snapshot.MetricTokens
)

// ParseQuotaMetric validates a quota metric.
func ParseQuotaMetric(metric string) (QuotaMetric, error) {
	if err := ValidateMetric(metric); err != nil {
		return "", err
	}
	return QuotaMetric(metric), nil
}

func (m QuotaMetric) String() string { return string(m) }

// QuotaScope is a typed quota scope type.
type QuotaScope string

const (
	ScopeOrganization QuotaScope = snapshot.ScopeOrganization
	ScopeTeam         QuotaScope = snapshot.ScopeTeam
	ScopeProject      QuotaScope = snapshot.ScopeProject
	ScopeUser         QuotaScope = snapshot.ScopeUser
	ScopeAPIKey       QuotaScope = snapshot.ScopeAPIKey
)

// ParseQuotaScope validates a scope type.
func ParseQuotaScope(scopeType string) (QuotaScope, error) {
	if err := ValidateScopeType(scopeType); err != nil {
		return "", err
	}
	return QuotaScope(scopeType), nil
}

func (s QuotaScope) String() string { return string(s) }

// IsOrganization reports whether the scope is org-wide.
func (s QuotaScope) IsOrganization() bool { return s == ScopeOrganization }
