package gatewayconfig

import (
	"errors"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestNewQuotaDefaultsWindow(t *testing.T) {
	t.Parallel()
	q, err := NewQuota("quota_1", "org_1", snapshot.ScopeOrganization, "org_1", snapshot.MetricRequests, 10, "", time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if q.Window != snapshot.WindowTotal {
		t.Fatalf("window=%q", q.Window)
	}
}

func TestNewQuotaRejectsBadWindow(t *testing.T) {
	t.Parallel()
	_, err := NewQuota("quota_1", "org_1", snapshot.ScopeOrganization, "org_1", snapshot.MetricRequests, 10, "week", time.Now().UTC())
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestNewQuotaRejectsOrgScopeMismatch(t *testing.T) {
	t.Parallel()
	_, err := NewQuota("quota_1", "org_1", snapshot.ScopeOrganization, "org_other", snapshot.MetricRequests, 10, snapshot.WindowTotal, time.Now().UTC())
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateLimit(t *testing.T) {
	t.Parallel()
	if err := ValidateLimit(0); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
