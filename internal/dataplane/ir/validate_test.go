package ir_test

import (
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/ir"
)

func TestRejectOpenAITools(t *testing.T) {
	_, err := ir.DecodeOpenAIRequest([]byte(`{
		"model":"m",
		"messages":[{"role":"user","content":"hi"}],
		"tools":[{"type":"function","function":{"name":"x"}}]
	}`))
	if err == nil {
		t.Fatal("expected error")
	}
	u, ok := ir.AsUnsupported(err)
	if !ok || u.Feature != "tools" {
		t.Fatalf("err=%v", err)
	}
}

func TestRejectOpenAIVision(t *testing.T) {
	_, err := ir.DecodeOpenAIRequest([]byte(`{
		"model":"m",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"what"},
			{"type":"image_url","image_url":{"url":"https://x"}}
		]}]
	}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := ir.AsUnsupported(err); !ok {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "image_url") && !strings.Contains(err.Error(), "vision") {
		t.Fatalf("err=%v", err)
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

func TestRejectAnthropicImageBlock(t *testing.T) {
	_, err := ir.DecodeAnthropicRequest([]byte(`{
		"model":"m","max_tokens":64,
		"messages":[{"role":"user","content":[
			{"type":"image","source":{"type":"url","url":"https://x"}},
			{"type":"text","text":"desc"}
		]}]
	}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := ir.AsUnsupported(err); !ok {
		t.Fatalf("err=%v", err)
	}
}

func TestRejectAnthropicTools(t *testing.T) {
	_, err := ir.DecodeAnthropicRequest([]byte(`{
		"model":"m","max_tokens":64,
		"tools":[{"name":"x","input_schema":{}}],
		"messages":[{"role":"user","content":"hi"}]
	}`))
	if err == nil {
		t.Fatal("expected error")
	}
	u, ok := ir.AsUnsupported(err)
	if !ok || u.Feature != "tools" {
		t.Fatalf("err=%v", err)
	}
}
