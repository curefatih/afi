package dataplane

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestOpenAIDialectRoutesToBedrockProvider(t *testing.T) {
	stub := &stubBedrockChatProvider{
		chat: func(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
			if targetModel != "anthropic.claude-3-haiku" {
				t.Fatalf("target=%q", targetModel)
			}
			if len(req.Messages) == 0 || req.Messages[0].Content != "ping" {
				t.Fatalf("req=%+v", req)
			}
			return ir.ChatResult{
				StatusCode: http.StatusOK,
				Response: &ir.ChatResponse{
					ID: "chatcmpl-bedrock", Model: targetModel, Role: "assistant",
					Content: "pong", FinishReason: "stop",
					Usage: ir.Usage{PromptTokens: 1, CompletionTokens: 1},
				},
			}, nil
		},
	}
	p := bedrockProviderTestPipeline(stub)
	req := httptest.NewRequest(
		http.MethodPost,
		"/openai/v1/chat/completions",
		bytes.NewBufferString(`{"model":"bedrock-route","messages":[{"role":"user","content":"ping"}]}`),
	)
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	choices := out["choices"].([]any)
	msg := choices[0].(map[string]any)["message"].(map[string]any)
	if msg["content"] != "pong" {
		t.Fatalf("response=%v", out)
	}
}

type stubBedrockChatProvider struct {
	chat func(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error)
}

func (s *stubBedrockChatProvider) Type() string { return "bedrock" }
func (s *stubBedrockChatProvider) Capabilities() ProviderCaps {
	return ProviderCaps{Chat: true, Stream: true}
}
func (s *stubBedrockChatProvider) Chat(context.Context, snapshot.Provider, string, []byte, bool) (*http.Response, error) {
	return nil, nil
}
func (s *stubBedrockChatProvider) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	return s.chat(ctx, provider, targetModel, req)
}

func bedrockProviderTestPipeline(stub *stubBedrockChatProvider) *Pipeline {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov-bedrock", Type: "bedrock",
			BaseURL: "https://bedrock-runtime.us-west-2.amazonaws.com",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "bedrock-route",
			ProviderID: "prov-bedrock", TargetModel: "anthropic.claude-3-haiku",
		}},
	}))
	reg := NewRegistry().Register(stub)
	return NewPipeline(holder, reg, slog.Default())
}
