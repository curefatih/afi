package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/curefatih/afi/internal/config"
)

type Provider struct {
	tracer         trace.Tracer
	requestCounter metric.Int64Counter
	tokenCounter   metric.Int64Counter
	latencyHist    metric.Float64Histogram
	shutdown       func(context.Context) error
}

func Init(ctx context.Context, cfg config.TelemetryConfig) (*Provider, error) {
	if !cfg.Enabled || cfg.OTLPEndpoint == "" {
		return &Provider{tracer: otel.Tracer("afi-gateway")}, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return nil, err
	}

	traceOpts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.OTLPEndpoint)}
	metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.OTLPEndpoint)}
	if cfg.Insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}
	metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		return nil, fmt.Errorf("otlp metric exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	meter := mp.Meter("afi-gateway")
	reqCounter, err := meter.Int64Counter("afi.requests.total")
	if err != nil {
		return nil, err
	}
	tokenCounter, err := meter.Int64Counter("afi.tokens.total")
	if err != nil {
		return nil, err
	}
	latencyHist, err := meter.Float64Histogram("afi.request.duration_ms")
	if err != nil {
		return nil, err
	}

	return &Provider{
		tracer:         tp.Tracer("afi-gateway"),
		requestCounter: reqCounter,
		tokenCounter:   tokenCounter,
		latencyHist:    latencyHist,
		shutdown: func(ctx context.Context) error {
			_ = tp.Shutdown(ctx)
			return mp.Shutdown(ctx)
		},
	}, nil
}

func (p *Provider) Tracer() trace.Tracer {
	if p.tracer != nil {
		return p.tracer
	}
	return otel.Tracer("afi-gateway")
}

func (p *Provider) RecordRequest(ctx context.Context, model, provider string, statusCode int, latency time.Duration, tokens int) {
	attrs := []attribute.KeyValue{
		attribute.String("model", model),
		attribute.String("provider", provider),
		attribute.Int("status_code", statusCode),
	}
	if p.requestCounter != nil {
		p.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if p.tokenCounter != nil && tokens > 0 {
		p.tokenCounter.Add(ctx, int64(tokens), metric.WithAttributes(attrs...))
	}
	if p.latencyHist != nil {
		p.latencyHist.Record(ctx, float64(latency.Milliseconds()), metric.WithAttributes(attrs...))
	}
}

func (p *Provider) Shutdown(ctx context.Context) error {
	if p.shutdown != nil {
		return p.shutdown(ctx)
	}
	return nil
}
