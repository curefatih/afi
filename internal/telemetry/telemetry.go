// Package telemetry configures OpenTelemetry metrics and traces for AFI processes.
// Instrumentation uses the OTel API only; exporters are OTLP and optional Prometheus scrape.
package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Provider holds initialized OTel providers and an optional Prometheus scrape handler.
type Provider struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	MetricsHandler http.Handler // non-nil when Prometheus scrape is enabled
	shutdowns      []func(context.Context) error
}

// Shutdown flushes and stops exporters.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	var first error
	for i := len(p.shutdowns) - 1; i >= 0; i-- {
		if err := p.shutdowns[i](ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Init configures global TracerProvider and MeterProvider from kernel telemetry config.
// serviceName is used when cfg.Telemetry.ServiceName is empty (e.g. "afi-gateway").
// When telemetry is disabled, returns a no-op Provider and nil error.
func Init(ctx context.Context, cfg *kernel.Config, serviceName string) (*Provider, error) {
	if cfg == nil || !cfg.Telemetry.Enabled {
		return &Provider{}, nil
	}

	name := strings.TrimSpace(cfg.Telemetry.ServiceName)
	if name == "" {
		name = strings.TrimSpace(serviceName)
	}
	if name == "" {
		name = "afi"
	}

	res, err := newResource(ctx, name, cfg.Telemetry.Environment)
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	p := &Provider{}

	tp, err := newTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, err
	}
	if tp != nil {
		p.TracerProvider = tp
		otel.SetTracerProvider(tp)
		p.shutdowns = append(p.shutdowns, tp.Shutdown)
	}

	mp, metricsHandler, err := newMeterProvider(ctx, cfg, res)
	if err != nil {
		_ = p.Shutdown(ctx)
		return nil, err
	}
	if mp != nil {
		p.MeterProvider = mp
		otel.SetMeterProvider(mp)
		p.shutdowns = append(p.shutdowns, mp.Shutdown)
	}
	p.MetricsHandler = metricsHandler

	return p, nil
}

func newResource(ctx context.Context, serviceName, environment string) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{semconv.ServiceName(serviceName)}
	if environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironment(environment))
	}
	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(attrs...),
	)
}

func newTracerProvider(ctx context.Context, cfg *kernel.Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	endpoint := strings.TrimSpace(cfg.Telemetry.OTLPEndpoint)
	if endpoint == "" {
		return sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(buildSampler(cfg)),
		), nil
	}

	exporter, err := newOTLPTraceExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(buildSampler(cfg)),
		sdktrace.WithBatcher(exporter),
	), nil
}

func buildSampler(cfg *kernel.Config) sdktrace.Sampler {
	switch cfg.Telemetry.TracesSampler {
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Telemetry.TracesSamplerArg))
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

func newMeterProvider(ctx context.Context, cfg *kernel.Config, res *resource.Resource) (*sdkmetric.MeterProvider, http.Handler, error) {
	var readers []sdkmetric.Reader
	var metricsHandler http.Handler

	if cfg.Telemetry.MetricsPrometheus {
		reg := prometheus.NewRegistry()
		exporter, err := otelprom.New(otelprom.WithRegisterer(reg))
		if err != nil {
			return nil, nil, fmt.Errorf("prometheus exporter: %w", err)
		}
		readers = append(readers, exporter)
		metricsHandler = promhttp.HandlerFor(reg, promhttp.HandlerOpts{EnableOpenMetrics: true})
	}

	endpoint := strings.TrimSpace(cfg.Telemetry.OTLPEndpoint)
	if endpoint != "" {
		exp, err := newOTLPMetricExporter(ctx, cfg)
		if err != nil {
			return nil, nil, err
		}
		readers = append(readers, sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(15*time.Second)))
	}

	if len(readers) == 0 {
		return sdkmetric.NewMeterProvider(sdkmetric.WithResource(res)), nil, nil
	}

	opts := []sdkmetric.Option{sdkmetric.WithResource(res)}
	for _, r := range readers {
		opts = append(opts, sdkmetric.WithReader(r))
	}
	return sdkmetric.NewMeterProvider(opts...), metricsHandler, nil
}

func newOTLPTraceExporter(ctx context.Context, cfg *kernel.Config) (sdktrace.SpanExporter, error) {
	endpoint, urlPath, insecure := normalizeOTLPEndpoint(cfg.Telemetry.OTLPEndpoint, cfg.Telemetry.OTLPInsecure)
	headers := parseOTLPHeaders(cfg.Telemetry.OTLPHeaders)

	switch cfg.Telemetry.OTLPProtocol {
	case "grpc":
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
		if insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(headers))
		}
		return otlptracegrpc.New(ctx, opts...)
	default:
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
		if urlPath != "" {
			opts = append(opts, otlptracehttp.WithURLPath(urlPath))
		}
		if insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(headers))
		}
		return otlptracehttp.New(ctx, opts...)
	}
}

func newOTLPMetricExporter(ctx context.Context, cfg *kernel.Config) (sdkmetric.Exporter, error) {
	endpoint, urlPath, insecure := normalizeOTLPEndpoint(cfg.Telemetry.OTLPEndpoint, cfg.Telemetry.OTLPInsecure)
	headers := parseOTLPHeaders(cfg.Telemetry.OTLPHeaders)

	switch cfg.Telemetry.OTLPProtocol {
	case "grpc":
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
		if insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(headers))
		}
		return otlpmetricgrpc.New(ctx, opts...)
	default:
		opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(endpoint)}
		if urlPath != "" {
			// Metric path: if user gave /v1/traces, swap; else use as-is or default.
			metricPath := urlPath
			if strings.HasSuffix(metricPath, "/v1/traces") {
				metricPath = strings.TrimSuffix(metricPath, "/v1/traces") + "/v1/metrics"
			}
			opts = append(opts, otlpmetrichttp.WithURLPath(metricPath))
		}
		if insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(headers))
		}
		return otlpmetrichttp.New(ctx, opts...)
	}
}

// normalizeOTLPEndpoint accepts host:port, http(s)://host:port, or full URL with path.
func normalizeOTLPEndpoint(raw string, insecureFlag bool) (endpoint, urlPath string, insecure bool) {
	raw = strings.TrimSpace(raw)
	insecure = insecureFlag
	if raw == "" {
		return "", "", insecure
	}
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "https://"):
		insecure = false
		raw = raw[len("https://"):]
	case strings.HasPrefix(lower, "http://"):
		insecure = true
		raw = raw[len("http://"):]
	}
	if i := strings.Index(raw, "/"); i >= 0 {
		endpoint = raw[:i]
		urlPath = raw[i:]
	} else {
		endpoint = raw
	}
	return endpoint, urlPath, insecure
}

func parseOTLPHeaders(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// Meter returns a named meter from the global provider.
func Meter(name string) metric.Meter {
	return otel.Meter(name)
}

// HTTPHandler wraps h with OTel HTTP server instrumentation.
func HTTPHandler(h http.Handler, operation string) http.Handler {
	if h == nil {
		return nil
	}
	if operation == "" {
		operation = "http"
	}
	return otelhttp.NewHandler(h, operation)
}

// HTTPClient returns an *http.Client whose transport propagates W3C trace context.
func HTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
}

// WrapTransport returns an instrumented RoundTripper.
func WrapTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base)
}

// MountMetrics registers /metrics when handler is non-nil.
func MountMetrics(mux *http.ServeMux, handler http.Handler) {
	if mux == nil || handler == nil {
		return
	}
	mux.Handle("/metrics", handler)
}
