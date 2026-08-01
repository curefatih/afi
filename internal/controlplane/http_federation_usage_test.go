package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/usage"
)

func TestHandleFederationUsageReportsRequiresRegionalMode(t *testing.T) {
	cfg := testCfg()
	cfg.Federation.Mode = "home"
	cfg.Federation.JoinToken = "join-secret"
	api := &fakePlatform{}
	s := &Server{
		cfg:       cfg,
		config:    api,
		app:       platform.New(api, &fakePublisher{}),
		publisher: &fakePublisher{},
	}

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/federation/usage-reports", nil)
	req.Header.Set("X-AFI-Federation-Token", "join-secret")
	rec := httptest.NewRecorder()
	s.handleFederationUsageReports(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleFederationUsageReportsAuthAndList(t *testing.T) {
	cfg := testCfg()
	cfg.Federation.Mode = "regional"
	cfg.Federation.JoinToken = "join-secret"
	now := time.Now().UTC().Truncate(time.Second)
	api := &fakePlatform{
		usageReports: []UsageEvent{{
			ID: 1, OrganizationID: "org_1", Model: "gpt-test", Status: "ok",
			CreatedAt: now, Tags: map[string]string{"region": "eu-west"},
		}},
	}
	s := &Server{
		cfg:       cfg,
		config:    api,
		app:       platform.New(api, &fakePublisher{}),
		publisher: &fakePublisher{},
	}

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/federation/usage-reports?limit=10", nil)
	rec := httptest.NewRecorder()
	s.handleFederationUsageReports(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing token want 401 got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/internal/v1/federation/usage-reports?limit=10", nil)
	req.Header.Set("X-AFI-Federation-Token", "wrong")
	rec = httptest.NewRecorder()
	s.handleFederationUsageReports(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad token want 401 got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/internal/v1/federation/usage-reports?limit=10", nil)
	req.Header.Set("X-AFI-Federation-Token", "join-secret")
	rec = httptest.NewRecorder()
	s.handleFederationUsageReports(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Reports []usage.Record `json:"reports"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Reports) != 1 || out.Reports[0].ID != 1 {
		t.Fatalf("reports=%+v", out.Reports)
	}
}

func TestHandleFederationUsageReportsBadSince(t *testing.T) {
	cfg := testCfg()
	cfg.Federation.Mode = "regional"
	cfg.Federation.JoinToken = "join-secret"
	api := &fakePlatform{}
	s := &Server{
		cfg:       cfg,
		config:    api,
		app:       platform.New(api, &fakePublisher{}),
		publisher: &fakePublisher{},
	}

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/federation/usage-reports?since=not-a-time", nil)
	req.Header.Set("X-AFI-Federation-Token", "join-secret")
	rec := httptest.NewRecorder()
	s.handleFederationUsageReports(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", rec.Code)
	}
}
