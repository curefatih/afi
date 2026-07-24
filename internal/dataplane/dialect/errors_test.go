package dialect_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/dialect"
	"github.com/curefatih/afi/internal/dataplane/ir"
)

func TestParseErrorBodyOpenAI(t *testing.T) {
	msg, typ := dialect.ParseErrorBody([]byte(`{"error":{"message":"bad model","type":"invalid_request_error","code":"model_not_found"}}`))
	if msg != "bad model" || typ != "invalid_request_error" {
		t.Fatalf("msg=%q typ=%q", msg, typ)
	}
}

func TestParseErrorBodyAnthropic(t *testing.T) {
	msg, typ := dialect.ParseErrorBody([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`))
	if msg != "slow down" || typ != "rate_limit_error" {
		t.Fatalf("msg=%q typ=%q", msg, typ)
	}
}

func TestParseErrorBodyGemini(t *testing.T) {
	msg, typ := dialect.ParseErrorBody([]byte(`{"error":{"code":400,"message":"API key not valid","status":"INVALID_ARGUMENT"}}`))
	if msg != "API key not valid" || typ != "INVALID_ARGUMENT" {
		t.Fatalf("msg=%q typ=%q", msg, typ)
	}
}

func TestParseErrorBodyPlainText(t *testing.T) {
	msg, typ := dialect.ParseErrorBody([]byte("upstream exploded"))
	if msg != "upstream exploded" || typ != "" {
		t.Fatalf("msg=%q typ=%q", msg, typ)
	}
}

func TestWriteUpstreamErrorRemapsToAnthropic(t *testing.T) {
	rr := httptest.NewRecorder()
	openaiBody := []byte(`{"error":{"message":"model not found","type":"invalid_request_error"}}`)
	dialect.WriteUpstreamError(rr, ir.DialectAnthropic, http.StatusBadRequest, openaiBody)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["type"] != "error" {
		t.Fatalf("envelope=%v", out)
	}
	errObj := out["error"].(map[string]any)
	if errObj["message"] != "model not found" || errObj["type"] != "invalid_request_error" {
		t.Fatalf("error=%v", errObj)
	}
	if strings.Contains(rr.Body.String(), "chat.completion") {
		t.Fatal("should not leak openai object field")
	}
}

func TestWriteUpstreamErrorRemapsToOpenAI(t *testing.T) {
	rr := httptest.NewRecorder()
	anthBody := []byte(`{"type":"error","error":{"type":"api_error","message":"overloaded"}}`)
	dialect.WriteUpstreamError(rr, ir.DialectOpenAI, http.StatusInternalServerError, anthBody)
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	errObj := out["error"].(map[string]any)
	if errObj["message"] != "overloaded" {
		t.Fatalf("error=%v", errObj)
	}
	// Anthropic api_error → OpenAI server_error
	if errObj["type"] != "server_error" {
		t.Fatalf("type=%v", errObj["type"])
	}
}

func TestWriteErrorAnthropicNormalizesQuota(t *testing.T) {
	rr := httptest.NewRecorder()
	dialect.WriteError(rr, ir.DialectAnthropic, http.StatusTooManyRequests, "quota exceeded", "insufficient_quota")
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	errObj := out["error"].(map[string]any)
	if errObj["type"] != "rate_limit_error" {
		t.Fatalf("type=%v body=%s", errObj["type"], rr.Body.String())
	}
}

func TestWriteErrorGeminiUsesHTTPStatusVocabulary(t *testing.T) {
	rr := httptest.NewRecorder()
	dialect.WriteError(
		rr, ir.DialectGemini, http.StatusUnauthorized,
		"invalid api key", "invalid_request_error",
	)
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	errObj := out["error"].(map[string]any)
	if errObj["status"] != "UNAUTHENTICATED" {
		t.Fatalf("status=%v body=%s", errObj["status"], rr.Body.String())
	}
	if errObj["code"] != float64(http.StatusUnauthorized) {
		t.Fatalf("code=%v body=%s", errObj["code"], rr.Body.String())
	}
}
