package gatewayconfig

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestParseQuotaValueObjects(t *testing.T) {
	t.Parallel()
	w, err := ParseQuotaWindow("")
	if err != nil || w != WindowTotal {
		t.Fatalf("window=%q err=%v", w, err)
	}
	m, err := ParseQuotaMetric(snapshot.MetricTokens)
	if err != nil || m != MetricTokens {
		t.Fatalf("metric=%q err=%v", m, err)
	}
	s, err := ParseQuotaScope(snapshot.ScopeOrganization)
	if err != nil || !s.IsOrganization() {
		t.Fatalf("scope=%q err=%v", s, err)
	}
	if _, err := ParseQuotaMetric("bytes"); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
