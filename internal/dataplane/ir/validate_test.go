package ir_test

import (
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/ir"
)

func TestAllowOpenAITools(t *testing.T) {
	req, err := ir.DecodeOpenAIRequest([]byte(`{
		"model":"m",
		"messages":[{"role":"user","content":"hi"}],
		"tools":[{"type":"function","function":{"name":"x","description":"d","parameters":{"type":"object"}}}],
		"tool_choice":"auto"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Tools) != 1 || req.Tools[0].Name != "x" {
		t.Fatalf("tools=%+v", req.Tools)
	}
	if req.ToolChoice == nil || req.ToolChoice.Mode != ir.ToolChoiceAuto {
		t.Fatalf("tool_choice=%+v", req.ToolChoice)
	}
}

func TestAllowOpenAIVision(t *testing.T) {
	req, err := ir.DecodeOpenAIRequest([]byte(`{
		"model":"m",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"what"},
			{"type":"image_url","image_url":{"url":"https://x/y.png"}}
		]}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Messages[0].Parts) != 2 {
		t.Fatalf("parts=%+v", req.Messages[0].Parts)
	}
	if req.Messages[0].Parts[1].Type != ir.ContentImage || req.Messages[0].Parts[1].Image.URL != "https://x/y.png" {
		t.Fatalf("image part=%+v", req.Messages[0].Parts[1])
	}
}

func TestAllowOpenAITextParts(t *testing.T) {
	req, err := ir.DecodeOpenAIRequest([]byte(`{
		"model":"m",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"hello"},
			{"type":"text","text":"world"}
		]}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(req.Messages[0].Content, "hello") || !strings.Contains(req.Messages[0].Content, "world") {
		t.Fatalf("%+v", req)
	}
	// Text-only parts should not populate Parts (keeps text-only providers simple).
	if len(req.Messages[0].Parts) != 0 {
		t.Fatalf("unexpected parts=%+v", req.Messages[0].Parts)
	}
}

func TestRejectOpenAIN(t *testing.T) {
	_, err := ir.DecodeOpenAIRequest([]byte(`{
		"model":"m","n":2,
		"messages":[{"role":"user","content":"hi"}]
	}`))
	if err == nil {
		t.Fatal("expected error")
	}
	u, ok := ir.AsUnsupported(err)
	if !ok || u.Feature != "n" {
		t.Fatalf("err=%v", err)
	}
}

func TestRejectOpenAILegacyFunctions(t *testing.T) {
	_, err := ir.DecodeOpenAIRequest([]byte(`{
		"model":"m",
		"functions":[{"name":"x"}],
		"messages":[{"role":"user","content":"hi"}]
	}`))
	if err == nil {
		t.Fatal("expected error")
	}
	u, ok := ir.AsUnsupported(err)
	if !ok || u.Feature != "functions" {
		t.Fatalf("err=%v", err)
	}
}

func TestRejectAnthropicThinking(t *testing.T) {
	_, err := ir.DecodeAnthropicRequest([]byte(`{
		"model":"m","max_tokens":64,
		"thinking":{"type":"enabled","budget_tokens":1024},
		"messages":[{"role":"user","content":"hi"}]
	}`))
	if err == nil {
		t.Fatal("expected error")
	}
	u, ok := ir.AsUnsupported(err)
	if !ok || u.Feature != "thinking" {
		t.Fatalf("err=%v", err)
	}
}

func TestAllowAnthropicImageBlock(t *testing.T) {
	req, err := ir.DecodeAnthropicRequest([]byte(`{
		"model":"m","max_tokens":64,
		"messages":[{"role":"user","content":[
			{"type":"image","source":{"type":"url","url":"https://x/y.png"}},
			{"type":"text","text":"desc"}
		]}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Messages[0].Parts) != 2 || req.Messages[0].Parts[0].Type != ir.ContentImage {
		t.Fatalf("parts=%+v", req.Messages[0].Parts)
	}
}

func TestAllowAnthropicTools(t *testing.T) {
	req, err := ir.DecodeAnthropicRequest([]byte(`{
		"model":"m","max_tokens":64,
		"tools":[{"name":"x","input_schema":{"type":"object"}}],
		"tool_choice":{"type":"any"},
		"messages":[{"role":"user","content":"hi"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Tools) != 1 || req.Tools[0].Name != "x" {
		t.Fatalf("tools=%+v", req.Tools)
	}
	if req.ToolChoice == nil || req.ToolChoice.Mode != ir.ToolChoiceRequired {
		t.Fatalf("tool_choice=%+v", req.ToolChoice)
	}
}

func TestRejectAnthropicDocumentBlock(t *testing.T) {
	_, err := ir.DecodeAnthropicRequest([]byte(`{
		"model":"m","max_tokens":64,
		"messages":[{"role":"user","content":[
			{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"AAA"}}
		]}]
	}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := ir.AsUnsupported(err); !ok {
		t.Fatalf("err=%v", err)
	}
}
