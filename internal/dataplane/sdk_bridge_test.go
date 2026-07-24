package dataplane

import (
	"context"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/sdk/chatir"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
)

type stubSDK struct {
	gotConfig sdkprovider.ProviderConfig
	gotTarget string
	gotReq    chatir.Request
}

func (*stubSDK) Type() string { return "stub_sdk" }
func (*stubSDK) Capabilities() sdkprovider.Capabilities {
	return sdkprovider.Capabilities{Chat: true, Embedding: true}
}

func (s *stubSDK) ChatIR(ctx context.Context, cfg sdkprovider.ProviderConfig, targetModel string, req chatir.Request) (chatir.Result, error) {
	_ = ctx
	s.gotConfig = cfg
	s.gotTarget = targetModel
	s.gotReq = req
	return chatir.Result{
		StatusCode: 200,
		Response: &chatir.Response{
			ID: "chatcmpl-stub", Model: targetModel, Role: "assistant", Content: "ok",
			FinishReason: "stop",
			Usage:        chatir.Usage{PromptTokens: 1, CompletionTokens: 1},
		},
	}, nil
}

func TestRegisterSDK(t *testing.T) {
	t.Parallel()
	reg := NewRegistry().RegisterSDK(&stubSDK{})
	adapter, ok := reg.Get("stub_sdk")
	if !ok {
		t.Fatal("missing")
	}
	caps := adapter.Capabilities()
	if !caps.Chat || !caps.Embedding || caps.Stream {
		t.Fatalf("caps=%+v", caps)
	}
}

func TestRegisterSDKRejectsByteChat(t *testing.T) {
	t.Parallel()
	reg := NewRegistry().RegisterSDK(&stubSDK{})
	adapter, ok := reg.Get("stub_sdk")
	if !ok {
		t.Fatal("missing")
	}
	_, err := adapter.Chat(context.Background(), snapshot.Provider{Type: "stub_sdk"}, "m", []byte(`{}`), false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ChatIR") {
		t.Fatalf("err=%v", err)
	}
}

func TestRegisterSDKChatIRBridge(t *testing.T) {
	t.Parallel()
	stub := &stubSDK{}
	reg := NewRegistry().RegisterSDK(stub)
	adapter, ok := reg.Get("stub_sdk")
	if !ok {
		t.Fatal("missing")
	}
	irp, ok := adapter.(IRChatProvider)
	if !ok {
		t.Fatal("expected IRChatProvider bridge")
	}
	provider := snapshot.Provider{
		ID: "prov", Type: "stub_sdk", Name: "Stub", BaseURL: "http://stub.local", APIKeyEnv: "STUB_KEY",
	}
	result, err := irp.ChatIR(context.Background(), provider, "m", ir.ChatRequest{
		Messages: []ir.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != 200 || result.Response == nil || result.Response.Content != "ok" {
		t.Fatalf("result=%+v", result)
	}
	if stub.gotTarget != "m" {
		t.Fatalf("target=%q", stub.gotTarget)
	}
	if len(stub.gotReq.Messages) != 1 || stub.gotReq.Messages[0].Content != "hi" {
		t.Fatalf("req=%+v", stub.gotReq)
	}
	if stub.gotConfig.ID != "prov" || stub.gotConfig.BaseURL != "http://stub.local" || stub.gotConfig.APIKeyEnv != "STUB_KEY" {
		t.Fatalf("config=%+v", stub.gotConfig)
	}
	if !stub.gotConfig.Capabilities.Chat {
		t.Fatalf("config caps=%+v", stub.gotConfig.Capabilities)
	}
}
