package echo

import (
	"context"
	"testing"

	"github.com/curefatih/afi/sdk/chatir"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
)

func TestEchoChatIR(t *testing.T) {
	t.Parallel()
	a := New()
	req := chatir.Request{
		Model: "echo-demo",
		Messages: []chatir.Message{
			{Role: "system", Content: "be nice"},
			{Role: "user", Content: "hello world"},
		},
	}
	res, err := a.ChatIR(context.Background(), sdkprovider.ProviderConfig{Type: Type}, "echo-demo", req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("status=%d", res.StatusCode)
	}
	if res.Response == nil {
		t.Fatal("expected response")
	}
	if res.Response.Content != "echo: hello world" {
		t.Fatalf("content=%q", res.Response.Content)
	}
	if res.Response.Role != "assistant" || res.Response.FinishReason != "stop" {
		t.Fatalf("%+v", res.Response)
	}
	if res.Response.Model != "echo-demo" {
		t.Fatalf("model=%q", res.Response.Model)
	}
	if res.Response.Usage.PromptTokens != 2 || res.Response.Usage.CompletionTokens != 3 {
		t.Fatalf("usage=%+v", res.Response.Usage)
	}
}

func TestEchoChatIRFallsBackToRequestModel(t *testing.T) {
	t.Parallel()
	req := chatir.Request{Model: "from-request", Messages: []chatir.Message{{Role: "user", Content: "hi"}}}
	res, err := New().ChatIR(context.Background(), sdkprovider.ProviderConfig{}, "", req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Response.Model != "from-request" {
		t.Fatalf("model=%q", res.Response.Model)
	}
}

func TestEchoChatIRWithoutUserMessage(t *testing.T) {
	t.Parallel()
	req := chatir.Request{Model: "m", Messages: []chatir.Message{{Role: "assistant", Content: "prior"}}}
	res, err := New().ChatIR(context.Background(), sdkprovider.ProviderConfig{}, "m", req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Response.Content != "echo: " {
		t.Fatalf("content=%q", res.Response.Content)
	}
	if res.Response.Usage.PromptTokens != 1 {
		t.Fatalf("prompt tokens=%d", res.Response.Usage.PromptTokens)
	}
}

func TestEchoRejectsStream(t *testing.T) {
	t.Parallel()
	_, err := New().ChatIR(context.Background(), sdkprovider.ProviderConfig{}, "m", chatir.Request{Stream: true})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEchoCapabilities(t *testing.T) {
	t.Parallel()
	a := New()
	if a.Type() != Type {
		t.Fatalf("type=%q", a.Type())
	}
	caps := a.Capabilities()
	if !caps.Chat || caps.Stream {
		t.Fatalf("caps=%+v", caps)
	}
}
