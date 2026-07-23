package telemetry

import (
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

func TestInitDisabled(t *testing.T) {
	p, err := Init(t.Context(), &kernel.Config{}, "afi-test")
	if err != nil {
		t.Fatal(err)
	}
	if p.MetricsHandler != nil {
		t.Fatal("expected nil metrics handler when disabled")
	}
	if err := p.Shutdown(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestInitPrometheusOnly(t *testing.T) {
	cfg := &kernel.Config{}
	cfg.Telemetry.Enabled = true
	cfg.Telemetry.MetricsPrometheus = true
	cfg.Telemetry.OTLPProtocol = "http"
	cfg.Telemetry.TracesSampler = "parentbased_always_on"
	cfg.Telemetry.TracesSamplerArg = 1

	p, err := Init(t.Context(), cfg, "afi-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Shutdown(t.Context()) }()
	if p.MetricsHandler == nil {
		t.Fatal("expected prometheus metrics handler")
	}
	g, err := NewGatewayMetrics()
	if err != nil {
		t.Fatal(err)
	}
	g.RecordRequest(t.Context(), "chat", OutcomeOK, 0.01)
	g.SetSnapshotVersion(42)
}

func TestNormalizeOTLPEndpoint(t *testing.T) {
	cases := []struct {
		raw          string
		insecureFlag bool
		wantEndpoint string
		wantPath     string
		wantInsecure bool
	}{
		{"127.0.0.1:4318", true, "127.0.0.1:4318", "", true},
		{"http://127.0.0.1:4318", false, "127.0.0.1:4318", "", true},
		{"https://otel.example:4318", true, "otel.example:4318", "", false},
		{"http://127.0.0.1:4318/v1/traces", false, "127.0.0.1:4318", "/v1/traces", true},
	}
	for _, tc := range cases {
		ep, path, insecure := normalizeOTLPEndpoint(tc.raw, tc.insecureFlag)
		if ep != tc.wantEndpoint || path != tc.wantPath || insecure != tc.wantInsecure {
			t.Fatalf("%q: got (%q,%q,%v) want (%q,%q,%v)",
				tc.raw, ep, path, insecure, tc.wantEndpoint, tc.wantPath, tc.wantInsecure)
		}
	}
}

func TestParseOTLPHeaders(t *testing.T) {
	h := parseOTLPHeaders("api-key=secret, x-scope=afi")
	if h["api-key"] != "secret" || h["x-scope"] != "afi" {
		t.Fatalf("got %#v", h)
	}
}
