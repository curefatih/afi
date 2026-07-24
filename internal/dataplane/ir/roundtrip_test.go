package ir_test

import (
	"encoding/json"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/dialect"
	"github.com/curefatih/afi/internal/dataplane/ir"
)

func TestOpenAIAnthropicRoundTrip(t *testing.T) {
	openaiBody := []byte(`{
		"model":"gpt-x",
		"messages":[
			{"role":"system","content":"be brief"},
			{"role":"user","content":"hi"}
		],
		"max_tokens":100,
		"temperature":0.2
	}`)
	req, err := ir.DecodeOpenAIRequest(openaiBody)
	if err != nil {
		t.Fatal(err)
	}
	if req.System != "be brief" || len(req.Messages) != 1 || req.Messages[0].Content != "hi" {
		t.Fatalf("req=%+v", req)
	}

	anthBody, err := ir.EncodeAnthropic(req, "claude-x")
	if err != nil {
		t.Fatal(err)
	}
	var anth map[string]any
	if err := json.Unmarshal(anthBody, &anth); err != nil {
		t.Fatal(err)
	}
	if anth["system"] != "be brief" || anth["model"] != "claude-x" {
		t.Fatalf("anth=%v", anth)
	}

	back, err := ir.DecodeAnthropicRequest(anthBody)
	if err != nil {
		t.Fatal(err)
	}
	if back.System != "be brief" || back.Messages[0].Content != "hi" {
		t.Fatalf("back=%+v", back)
	}

	oa2, err := ir.EncodeOpenAI(back)
	if err != nil {
		t.Fatal(err)
	}
	var oa map[string]any
	_ = json.Unmarshal(oa2, &oa)
	msgs := oa["messages"].([]any)
	if len(msgs) < 2 {
		t.Fatalf("expected system+user, got %v", oa)
	}
}

func TestResponseRoundTrip(t *testing.T) {
	resp := ir.ChatResponse{
		ID: "msg_1", Model: "m", Role: "assistant", Content: "hello",
		FinishReason: "stop",
		Usage:        ir.Usage{PromptTokens: 3, CompletionTokens: 2},
	}
	anth, err := ir.EncodeAnthropicResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ir.DecodeAnthropicResponse(anth)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Content != "hello" || decoded.Usage.PromptTokens != 3 {
		t.Fatalf("%+v", decoded)
	}
	oa, err := ir.EncodeOpenAIResponse(decoded)
	if err != nil {
		t.Fatal(err)
	}
	decoded2, err := ir.DecodeOpenAIResponse(oa)
	if err != nil {
		t.Fatal(err)
	}
	if decoded2.Content != "hello" || decoded2.FinishReason != "stop" {
		t.Fatalf("%+v", decoded2)
	}
}

func TestDialectCodecs(t *testing.T) {
	oa, err := dialect.For(ir.DialectOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	anth, err := dialect.For(ir.DialectAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	req, err := oa.DecodeRequest([]byte(`{"model":"m","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	body, err := anth.EncodeResponse(ir.ChatResponse{ID: "1", Model: "m", Content: "y", FinishReason: "stop"})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(body) {
		t.Fatal(string(body))
	}
	_ = req
}
