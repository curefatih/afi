package controlplane

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFederationRegionalReadOnly(t *testing.T) {
	cfg := testCfg()
	cfg.Federation.Mode = "regional"
	s := &Server{cfg: cfg}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	h := s.federationRegionalReadOnly(inner)

	getOK := httptest.NewRequest(http.MethodGet, "/api/v1/platform/regions", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, getOK)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET platform: %d", rec.Code)
	}

	postAuth := httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/login", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, postAuth)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST auth: %d", rec.Code)
	}

	postRegion := httptest.NewRequest(http.MethodPost, "/api/v1/platform/regions", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, postRegion)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST region want 403 got %d body=%s", rec.Code, rec.Body.String())
	}

	export := httptest.NewRequest(http.MethodGet, "/internal/v1/federation/regions/eu-west/export", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, export)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("GET federation export want 403 got %d", rec.Code)
	}

	usageReports := httptest.NewRequest(http.MethodGet, "/internal/v1/federation/usage-reports", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, usageReports)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET usage-reports want 200 got %d", rec.Code)
	}

	cfg.Federation.Mode = "home"
	h = s.federationRegionalReadOnly(inner)
	postRegion = httptest.NewRequest(http.MethodPost, "/api/v1/platform/regions", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, postRegion)
	if rec.Code != http.StatusOK {
		t.Fatalf("home mode should allow mutations: %d", rec.Code)
	}
}
