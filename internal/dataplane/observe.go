package dataplane

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func modalityFromPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/v1/chat/completions"):
		return ModalityChat
	case strings.HasPrefix(path, "/v1/messages"):
		return ModalityMessages
	case strings.HasPrefix(path, "/v1/embeddings"):
		return ModalityEmbedding
	case strings.HasPrefix(path, "/v1/audio/speech"):
		return ModalityTTS
	case strings.HasPrefix(path, "/v1/audio/transcriptions"):
		return ModalitySTT
	case strings.HasPrefix(path, "/mcp/"):
		return ModalityMCP
	case strings.HasPrefix(path, "/a2a/"):
		return ModalityA2A
	default:
		return "other"
	}
}

func outcomeFromStatus(code int) string {
	switch {
	case code == http.StatusUnauthorized:
		return telemetry.OutcomeAuth
	case code == http.StatusTooManyRequests:
		return telemetry.OutcomeQuota
	case code == http.StatusForbidden:
		return telemetry.OutcomeDeny
	case code >= 500:
		return telemetry.OutcomeError
	case code >= 400:
		return telemetry.OutcomeError
	default:
		return telemetry.OutcomeOK
	}
}

func (p *Pipeline) withGatewayMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if p.Metrics == nil || path == "/healthz" || path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		p.Metrics.RecordRequest(r.Context(), modalityFromPath(path), outcomeFromStatus(rec.code), time.Since(start).Seconds())
	})
}

func startPipelineSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return telemetry.Tracer("afi.gateway").Start(ctx, name, trace.WithAttributes(attrs...))
}

func spanError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
