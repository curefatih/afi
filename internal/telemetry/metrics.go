package telemetry

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Attr keys used across AFI instrumentation (low cardinality on metrics).
const (
	AttrModality     = "afi.modality"
	AttrOutcome      = "afi.outcome"
	AttrProviderType = "afi.provider_type"
	AttrOrgID        = "afi.org_id"
	AttrRouteModel   = "afi.route_model"
	AttrTokenType    = "afi.token_type"
	AttrStatusClass  = "http.status_class"
)

// Outcomes for afi.gateway.requests.
const (
	OutcomeOK    = "ok"
	OutcomeError = "error"
	OutcomeQuota = "quota"
	OutcomeAuth  = "auth"
	OutcomeDeny  = "deny"
)

// GatewayMetrics holds gateway instruments.
type GatewayMetrics struct {
	Requests         metric.Int64Counter
	RequestDuration  metric.Float64Histogram
	UpstreamDuration metric.Float64Histogram
	Tokens           metric.Int64Counter
	QuotaRejections  metric.Int64Counter
	snapshotVersion  atomic.Int64
}

// NewGatewayMetrics creates instruments on the global meter provider.
func NewGatewayMetrics() (*GatewayMetrics, error) {
	m := Meter("afi.gateway")
	g := &GatewayMetrics{}
	var err error
	if g.Requests, err = m.Int64Counter("afi.gateway.requests",
		metric.WithDescription("Gateway requests by modality and outcome")); err != nil {
		return nil, err
	}
	if g.RequestDuration, err = m.Float64Histogram("afi.gateway.request_duration",
		metric.WithDescription("End-to-end gateway request duration"),
		metric.WithUnit("s")); err != nil {
		return nil, err
	}
	if g.UpstreamDuration, err = m.Float64Histogram("afi.gateway.upstream_duration",
		metric.WithDescription("Upstream provider/proxy duration"),
		metric.WithUnit("s")); err != nil {
		return nil, err
	}
	if g.Tokens, err = m.Int64Counter("afi.gateway.tokens",
		metric.WithDescription("Tokens observed on successful chat/messages calls")); err != nil {
		return nil, err
	}
	if g.QuotaRejections, err = m.Int64Counter("afi.gateway.quota_rejections",
		metric.WithDescription("Requests rejected for insufficient quota")); err != nil {
		return nil, err
	}
	_, err = m.Int64ObservableGauge("afi.gateway.snapshot_version",
		metric.WithDescription("Currently loaded gateway snapshot version"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(g.snapshotVersion.Load())
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// SetSnapshotVersion updates the observable gauge.
func (g *GatewayMetrics) SetSnapshotVersion(v int64) {
	if g != nil {
		g.snapshotVersion.Store(v)
	}
}

// RecordRequest increments request counters and duration.
func (g *GatewayMetrics) RecordRequest(ctx context.Context, modality, outcome string, seconds float64) {
	if g == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String(AttrModality, modality),
		attribute.String(AttrOutcome, outcome),
	)
	g.Requests.Add(ctx, 1, attrs)
	if seconds >= 0 {
		g.RequestDuration.Record(ctx, seconds, attrs)
	}
	if outcome == OutcomeQuota {
		g.QuotaRejections.Add(ctx, 1, metric.WithAttributes(attribute.String(AttrModality, modality)))
	}
}

// RecordUpstream records upstream call duration.
func (g *GatewayMetrics) RecordUpstream(ctx context.Context, modality, providerType string, seconds float64) {
	if g == nil || seconds < 0 {
		return
	}
	g.UpstreamDuration.Record(ctx, seconds, metric.WithAttributes(
		attribute.String(AttrModality, modality),
		attribute.String(AttrProviderType, providerType),
	))
}

// RecordTokens increments token counters.
func (g *GatewayMetrics) RecordTokens(ctx context.Context, modality string, prompt, completion int64) {
	if g == nil {
		return
	}
	if prompt > 0 {
		g.Tokens.Add(ctx, prompt, metric.WithAttributes(
			attribute.String(AttrModality, modality),
			attribute.String(AttrTokenType, "prompt"),
		))
	}
	if completion > 0 {
		g.Tokens.Add(ctx, completion, metric.WithAttributes(
			attribute.String(AttrModality, modality),
			attribute.String(AttrTokenType, "completion"),
		))
	}
}

// ControlPlaneMetrics holds control-plane HTTP instruments.
type ControlPlaneMetrics struct {
	Requests metric.Int64Counter
	Duration metric.Float64Histogram
}

// NewControlPlaneMetrics creates CP instruments.
func NewControlPlaneMetrics() (*ControlPlaneMetrics, error) {
	m := Meter("afi.controlplane")
	c := &ControlPlaneMetrics{}
	var err error
	if c.Requests, err = m.Int64Counter("afi.controlplane.http_requests",
		metric.WithDescription("Control plane HTTP requests")); err != nil {
		return nil, err
	}
	if c.Duration, err = m.Float64Histogram("afi.controlplane.http_duration",
		metric.WithDescription("Control plane HTTP request duration"),
		metric.WithUnit("s")); err != nil {
		return nil, err
	}
	return c, nil
}

// Record records a CP HTTP request.
func (c *ControlPlaneMetrics) Record(ctx context.Context, route, statusClass string, seconds float64) {
	if c == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("http.route", route),
		attribute.String(AttrStatusClass, statusClass),
	)
	c.Requests.Add(ctx, 1, attrs)
	if seconds >= 0 {
		c.Duration.Record(ctx, seconds, attrs)
	}
}

// WorkerMetrics holds worker outbox instruments.
type WorkerMetrics struct {
	UsageProcessed          metric.Int64Counter
	UsageErrors             metric.Int64Counter
	PlatformEventsPublished metric.Int64Counter
	ProcessDuration         metric.Float64Histogram
}

// NewWorkerMetrics creates worker instruments.
func NewWorkerMetrics() (*WorkerMetrics, error) {
	m := Meter("afi.worker")
	w := &WorkerMetrics{}
	var err error
	if w.UsageProcessed, err = m.Int64Counter("afi.worker.usage_processed",
		metric.WithDescription("Usage outbox rows processed")); err != nil {
		return nil, err
	}
	if w.UsageErrors, err = m.Int64Counter("afi.worker.usage_errors",
		metric.WithDescription("Usage outbox processing errors")); err != nil {
		return nil, err
	}
	if w.PlatformEventsPublished, err = m.Int64Counter("afi.worker.platform_events_published",
		metric.WithDescription("Platform events published from outbox")); err != nil {
		return nil, err
	}
	if w.ProcessDuration, err = m.Float64Histogram("afi.worker.process_duration",
		metric.WithDescription("Worker ProcessOnce batch duration"),
		metric.WithUnit("s")); err != nil {
		return nil, err
	}
	return w, nil
}
