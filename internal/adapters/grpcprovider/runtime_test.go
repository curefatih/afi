package grpcprovider_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"testing"

	extensionv1 "github.com/curefatih/afi/gen/proto/afi/extension/v1"
	"github.com/curefatih/afi/internal/adapters/grpcprovider"
	"github.com/curefatih/afi/sdk/chatir"
	sdkhook "github.com/curefatih/afi/sdk/hook"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
	"google.golang.org/grpc"
)

type testPlugin struct {
	extensionv1.UnimplementedExtensionServer
	extensionv1.UnimplementedProviderServer
	extensionv1.UnimplementedProviderIRServer
	extensionv1.UnimplementedHookServer

	deny bool
}

func (p *testPlugin) Handshake(ctx context.Context, req *extensionv1.HandshakeRequest) (*extensionv1.HandshakeResponse, error) {
	return &extensionv1.HandshakeResponse{
		Name:         "testplugin",
		Version:      "0.1.0",
		ProviderType: "testgrpc",
		Capabilities: []extensionv1.Capability{
			extensionv1.Capability_CAPABILITY_PROVIDER_CHAT,
			extensionv1.Capability_CAPABILITY_PROVIDER_CHAT_IR,
			extensionv1.Capability_CAPABILITY_HOOK_BEFORE_CALL,
			extensionv1.Capability_CAPABILITY_HOOK_AFTER_CALL,
			extensionv1.Capability_CAPABILITY_HOOK_BEFORE_CHAT,
			extensionv1.Capability_CAPABILITY_HOOK_BEFORE_CHAT_IR,
			extensionv1.Capability_CAPABILITY_HOOK_AFTER_CHAT,
		},
	}, nil
}

func (p *testPlugin) Chat(ctx context.Context, req *extensionv1.ChatRequest) (*extensionv1.ChatResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"id": "chatcmpl-test",
		"choices": []map[string]any{{
			"message": map[string]string{"role": "assistant", "content": "grpc-ok"},
		}},
	})
	return &extensionv1.ChatResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json", "X-Test": "1"},
		Body:       body,
	}, nil
}

func (p *testPlugin) ChatIR(ctx context.Context, req *extensionv1.ChatIRRequest) (*extensionv1.ChatIRResponse, error) {
	_ = ctx
	irReq := grpcprovider.ChatIRRequestFromProto(req.GetRequest())
	text := ""
	for i := len(irReq.Messages) - 1; i >= 0; i-- {
		if irReq.Messages[i].Role == "user" {
			text = irReq.Messages[i].Content
			break
		}
	}
	model := req.GetTargetModel()
	if model == "" {
		model = irReq.Model
	}
	return grpcprovider.ChatIRResponseProto(chatir.Result{
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"application/json"}, "X-Test-IR": {"1"}},
		Response: &chatir.Response{
			ID: "chatcmpl-ir", Model: model, Role: "assistant",
			Content: "ir:" + text, FinishReason: "stop",
			Usage: chatir.Usage{PromptTokens: 1, CompletionTokens: 1},
		},
	}), nil
}

func (p *testPlugin) ChatIRStream(req *extensionv1.ChatIRRequest, stream extensionv1.ProviderIR_ChatIRStreamServer) error {
	_ = req
	return stream.Send(grpcprovider.ChatIRStreamEventProto(chatir.StreamEvent{
		Kind: chatir.StreamTextDelta, Text: "streamed",
	}))
}

func (p *testPlugin) BeforeCall(ctx context.Context, req *extensionv1.BeforeCallRequest) (*extensionv1.BeforeCallResponse, error) {
	call := req.GetCall()
	if call != nil {
		if call.RequestHeaders == nil {
			call.RequestHeaders = map[string]string{}
		}
		call.RequestHeaders["X-From-Hook"] = "yes"
		call.Tags = map[string]string{"k": "v"}
	}
	if p.deny {
		return &extensionv1.BeforeCallResponse{
			Decision: &extensionv1.CallDecision{Allow: false, Status: 403, Reason: "denied", Message: "nope"},
			Call:     call,
		}, nil
	}
	return &extensionv1.BeforeCallResponse{
		Decision: &extensionv1.CallDecision{Allow: true},
		Call:     call,
	}, nil
}

func (p *testPlugin) AfterCall(ctx context.Context, req *extensionv1.AfterCallRequest) (*extensionv1.AfterCallResponse, error) {
	return &extensionv1.AfterCallResponse{}, nil
}

func (p *testPlugin) BeforeChat(ctx context.Context, req *extensionv1.BeforeChatRequest) (*extensionv1.BeforeChatResponse, error) {
	return &extensionv1.BeforeChatResponse{Body: append([]byte("prefix:"), req.GetBody()...)}, nil
}

func (p *testPlugin) BeforeChatIR(ctx context.Context, req *extensionv1.BeforeChatIRRequest) (*extensionv1.BeforeChatIRResponse, error) {
	_ = ctx
	irReq := grpcprovider.ChatIRRequestFromProto(req.GetRequest())
	irReq.System = "typed-hook"
	return &extensionv1.BeforeChatIRResponse{Request: grpcprovider.ChatIRRequestProto(irReq)}, nil
}

func (p *testPlugin) AfterChat(ctx context.Context, req *extensionv1.AfterChatRequest) (*extensionv1.AfterChatResponse, error) {
	return &extensionv1.AfterChatResponse{}, nil
}

func servePlugin(t *testing.T, plugin *testPlugin) string {
	t.Helper()
	tcpLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	tcpSrv := grpc.NewServer()
	extensionv1.RegisterExtensionServer(tcpSrv, plugin)
	extensionv1.RegisterProviderServer(tcpSrv, plugin)
	extensionv1.RegisterProviderIRServer(tcpSrv, plugin)
	extensionv1.RegisterHookServer(tcpSrv, plugin)
	go func() { _ = tcpSrv.Serve(tcpLis) }()
	t.Cleanup(func() {
		tcpSrv.Stop()
		_ = tcpLis.Close()
	})
	return tcpLis.Addr().String()
}

func TestRuntimeDial(t *testing.T) {
	addr := servePlugin(t, &testPlugin{})
	ctx := context.Background()
	rt, err := grpcprovider.Start(ctx, []grpcprovider.Manifest{{
		ID:      "test",
		Address: addr,
	}}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close() })

	providers := rt.Providers()
	if len(providers) != 1 {
		t.Fatalf("providers=%d", len(providers))
	}
	if providers[0].Type() != "testgrpc" {
		t.Fatalf("type=%q", providers[0].Type())
	}

	resp, err := providers[0].Chat(ctx, sdkprovider.ProviderConfig{ID: "p1", Type: "testgrpc"}, "m", []byte(`{"messages":[]}`), false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if resp.Header.Get("X-Test") != "1" {
		t.Fatalf("headers=%v", resp.Header)
	}
	raw, _ := io.ReadAll(resp.Body)
	if !json.Valid(raw) {
		t.Fatalf("body=%s", raw)
	}

	irp, ok := providers[0].(sdkprovider.ChatIRProvider)
	if !ok {
		t.Fatal("expected ChatIRProvider")
	}
	irResult, err := irp.ChatIR(ctx, sdkprovider.ProviderConfig{ID: "p1", Type: "testgrpc"}, "m", chatir.Request{
		Messages: []chatir.Message{{Role: "user", Content: "ping"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if irResult.Response == nil || irResult.Response.Content != "ir:ping" {
		t.Fatalf("ir result=%+v", irResult)
	}
	if irResult.Header["X-Test-IR"][0] != "1" {
		t.Fatalf("ir headers=%v", irResult.Header)
	}

	var before sdkhook.BeforeCallHook
	var after sdkhook.AfterCallHook
	var beforeChat sdkhook.ChatHook
	var beforeChatIR sdkhook.ChatIRHook
	var afterChat sdkhook.AfterChatHook
	rt.ApplyHooks(func(h any) {
		switch v := h.(type) {
		case sdkhook.BeforeCallHook:
			before = v
		case sdkhook.AfterCallHook:
			after = v
		case sdkhook.ChatHook:
			beforeChat = v
		case sdkhook.ChatIRHook:
			beforeChatIR = v
		case sdkhook.AfterChatHook:
			afterChat = v
		}
	})
	if before == nil || after == nil || beforeChat == nil || beforeChatIR == nil || afterChat == nil {
		t.Fatalf("missing hooks before=%v after=%v beforeChat=%v beforeChatIR=%v afterChat=%v", before, after, beforeChat, beforeChatIR, afterChat)
	}

	call := &sdkhook.CallContext{Tags: map[string]string{}, RequestHeaders: map[string]string{}}
	dec, err := before.BeforeCall(ctx, call)
	if err != nil {
		t.Fatal(err)
	}
	if !dec.Allow {
		t.Fatalf("decision=%+v", dec)
	}
	if call.RequestHeaders["X-From-Hook"] != "yes" {
		t.Fatalf("request headers=%v", call.RequestHeaders)
	}
	if call.Tags["k"] != "v" {
		t.Fatalf("tags=%v", call.Tags)
	}

	if err := after.AfterCall(ctx, call, sdkhook.AfterCallInfo{Status: "ok", LatencyMs: 1}); err != nil {
		t.Fatal(err)
	}
	out, err := beforeChat.BeforeChat(ctx, []byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "prefix:hi" {
		t.Fatalf("beforeChat=%q", out)
	}
	irOut, err := beforeChatIR.BeforeChatIR(ctx, chatir.Request{
		Messages: []chatir.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if irOut.System != "typed-hook" {
		t.Fatalf("beforeChatIR=%+v", irOut)
	}
	if err := afterChat.AfterChat(ctx, sdkhook.AfterChatInfo{Model: "m", Status: "ok"}); err != nil {
		t.Fatal(err)
	}

	_, err = providers[0].Chat(ctx, sdkprovider.ProviderConfig{}, "m", nil, true)
	if err == nil {
		t.Fatal("expected stream error")
	}
}

func TestBeforeCallDeny(t *testing.T) {
	addr := servePlugin(t, &testPlugin{deny: true})
	rt, err := grpcprovider.Start(context.Background(), []grpcprovider.Manifest{{
		ID:      "deny",
		Address: addr,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rt.Close() })

	var before sdkhook.BeforeCallHook
	rt.ApplyHooks(func(h any) {
		if v, ok := h.(sdkhook.BeforeCallHook); ok {
			before = v
		}
	})
	dec, err := before.BeforeCall(context.Background(), &sdkhook.CallContext{})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allow || dec.Status != 403 || dec.Reason != "denied" {
		t.Fatalf("decision=%+v", dec)
	}
}

func TestManifestValidate(t *testing.T) {
	_, err := grpcprovider.Start(context.Background(), []grpcprovider.Manifest{{ID: "x"}}, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestProviderTypeOverride(t *testing.T) {
	addr := servePlugin(t, &testPlugin{})
	rt, err := grpcprovider.Start(context.Background(), []grpcprovider.Manifest{{
		ID:           "ovr",
		Address:      addr,
		ProviderType: "custom-type",
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rt.Close() })
	if rt.Providers()[0].Type() != "custom-type" {
		t.Fatalf("type=%q", rt.Providers()[0].Type())
	}
}
