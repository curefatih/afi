package ir_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/dialect"
	"github.com/curefatih/afi/internal/dataplane/ir"
)

func TestToolsRequestRoundTrip(t *testing.T) {
	openaiBody := []byte(`{
		"model":"gpt-x",
		"messages":[{"role":"user","content":"weather in SF?"}],
		"tools":[{"type":"function","function":{"name":"get_weather","description":"get it","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}}],
		"tool_choice":{"type":"function","function":{"name":"get_weather"}}
	}`)
	req, err := ir.DecodeOpenAIRequest(openaiBody)
	if err != nil {
		t.Fatal(err)
	}
	if req.ToolChoice == nil || req.ToolChoice.Mode != ir.ToolChoiceTool || req.ToolChoice.Name != "get_weather" {
		t.Fatalf("tool_choice=%+v", req.ToolChoice)
	}

	anthBody, err := ir.EncodeAnthropic(req, "claude-x")
	if err != nil {
		t.Fatal(err)
	}
	var anth map[string]any
	if err := json.Unmarshal(anthBody, &anth); err != nil {
		t.Fatal(err)
	}
	tools, ok := anth["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("anthropic tools=%v", anth["tools"])
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "get_weather" || tool["input_schema"] == nil {
		t.Fatalf("tool=%v", tool)
	}
	tc := anth["tool_choice"].(map[string]any)
	if tc["type"] != "tool" || tc["name"] != "get_weather" {
		t.Fatalf("tool_choice=%v", tc)
	}

	back, err := ir.DecodeAnthropicRequest(anthBody)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Tools) != 1 || back.Tools[0].Name != "get_weather" {
		t.Fatalf("back tools=%+v", back.Tools)
	}
	if back.ToolChoice == nil || back.ToolChoice.Mode != ir.ToolChoiceTool {
		t.Fatalf("back tool_choice=%+v", back.ToolChoice)
	}
}

func TestToolCallConversationRoundTrip(t *testing.T) {
	// assistant issues a tool call, tool responds, model summarizes.
	openaiBody := []byte(`{
		"model":"gpt-x",
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"SF\"}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"72F sunny"}
		]
	}`)
	req, err := ir.DecodeOpenAIRequest(openaiBody)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Messages) != 3 {
		t.Fatalf("messages=%+v", req.Messages)
	}
	if len(req.Messages[1].ToolCalls) != 1 || req.Messages[1].ToolCalls[0].ID != "call_1" {
		t.Fatalf("assistant tool calls=%+v", req.Messages[1])
	}
	if req.Messages[2].Role != "tool" || req.Messages[2].ToolCallID != "call_1" || req.Messages[2].Content != "72F sunny" {
		t.Fatalf("tool msg=%+v", req.Messages[2])
	}

	// IR -> Anthropic: tool_use in assistant turn, tool_result in a user turn.
	anthBody, err := ir.EncodeAnthropic(req, "claude-x")
	if err != nil {
		t.Fatal(err)
	}
	var anth struct {
		Messages []struct {
			Role    string           `json:"role"`
			Content []map[string]any `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(anthBody, &anth); err != nil {
		t.Fatal(err)
	}
	// user, assistant(tool_use), user(tool_result)
	if len(anth.Messages) != 3 {
		t.Fatalf("anth messages=%+v", anth.Messages)
	}
	if anth.Messages[1].Role != "assistant" || anth.Messages[1].Content[0]["type"] != "tool_use" {
		t.Fatalf("assistant turn=%+v", anth.Messages[1])
	}
	if anth.Messages[2].Role != "user" || anth.Messages[2].Content[0]["type"] != "tool_result" {
		t.Fatalf("tool_result turn=%+v", anth.Messages[2])
	}
	if anth.Messages[2].Content[0]["tool_use_id"] != "call_1" {
		t.Fatalf("tool_use_id=%v", anth.Messages[2].Content[0])
	}

	// Anthropic -> IR faithful.
	back, err := ir.DecodeAnthropicRequest(anthBody)
	if err != nil {
		t.Fatal(err)
	}
	var gotTool, gotResult bool
	for _, m := range back.Messages {
		if len(m.ToolCalls) > 0 && m.ToolCalls[0].ID == "call_1" {
			gotTool = true
		}
		if m.Role == "tool" && m.ToolCallID == "call_1" && strings.Contains(m.Content, "72F") {
			gotResult = true
		}
	}
	if !gotTool || !gotResult {
		t.Fatalf("round trip lost tool data: %+v", back.Messages)
	}
}

func TestVisionDataURLRoundTrip(t *testing.T) {
	dataURL := "data:image/png;base64,iVBORw0KGgo="
	openaiBody := []byte(`{
		"model":"gpt-x",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"describe"},
			{"type":"image_url","image_url":{"url":"` + dataURL + `"}}
		]}]
	}`)
	req, err := ir.DecodeOpenAIRequest(openaiBody)
	if err != nil {
		t.Fatal(err)
	}
	img := req.Messages[0].Parts[1].Image
	if img == nil || img.MediaType != "image/png" || img.Data != "iVBORw0KGgo=" {
		t.Fatalf("image=%+v", img)
	}

	// IR -> Anthropic base64 image block.
	anthBody, err := ir.EncodeAnthropic(req, "claude-x")
	if err != nil {
		t.Fatal(err)
	}
	var anth struct {
		Messages []struct {
			Content []map[string]any `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(anthBody, &anth); err != nil {
		t.Fatal(err)
	}
	block := anth.Messages[0].Content[1]
	if block["type"] != "image" {
		t.Fatalf("block=%v", block)
	}
	src := block["source"].(map[string]any)
	if src["type"] != "base64" || src["media_type"] != "image/png" || src["data"] != "iVBORw0KGgo=" {
		t.Fatalf("source=%v", src)
	}

	// Anthropic -> IR -> OpenAI restores a data URL.
	back, err := ir.DecodeAnthropicRequest(anthBody)
	if err != nil {
		t.Fatal(err)
	}
	oaBody, err := ir.EncodeOpenAI(back)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(oaBody), dataURL) {
		t.Fatalf("data url lost: %s", oaBody)
	}
}

func TestToolCallResponseRoundTrip(t *testing.T) {
	resp := ir.ChatResponse{
		ID: "msg_1", Model: "m", Role: "assistant",
		ToolCalls:    []ir.ToolCall{{ID: "call_1", Name: "get_weather", Arguments: `{"city":"SF"}`}},
		FinishReason: "tool_calls",
		Usage:        ir.Usage{PromptTokens: 5, CompletionTokens: 3},
	}
	anth, err := ir.EncodeAnthropicResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	var am map[string]any
	_ = json.Unmarshal(anth, &am)
	if am["stop_reason"] != "tool_use" {
		t.Fatalf("stop_reason=%v", am["stop_reason"])
	}
	decoded, err := ir.DecodeAnthropicResponse(anth)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.FinishReason != "tool_calls" || len(decoded.ToolCalls) != 1 || decoded.ToolCalls[0].Name != "get_weather" {
		t.Fatalf("decoded=%+v", decoded)
	}
	oa, err := ir.EncodeOpenAIResponse(decoded)
	if err != nil {
		t.Fatal(err)
	}
	var om map[string]any
	_ = json.Unmarshal(oa, &om)
	choice := om["choices"].([]any)[0].(map[string]any)
	if choice["finish_reason"] != "tool_calls" {
		t.Fatalf("finish=%v", choice)
	}
	msg := choice["message"].(map[string]any)
	if msg["tool_calls"] == nil {
		t.Fatalf("no tool_calls in openai response: %v", msg)
	}
}

func TestStreamToolCallsOpenAIToAnthropic(t *testing.T) {
	// OpenAI SSE with a streamed tool call, translated to Anthropic events.
	sse := strings.Join([]string{
		`data: {"id":"c1","model":"m","choices":[{"delta":{"role":"assistant"}}]}`,
		`data: {"id":"c1","model":"m","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"get_weather","arguments":""}}]}}]}`,
		`data: {"id":"c1","model":"m","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]}}]}`,
		`data: {"id":"c1","model":"m","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"SF\"}"}}]}}]}`,
		`data: {"id":"c1","model":"m","choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
		"",
	}, "\n")

	events := dialect.ParseOpenAISSE(strings.NewReader(sse))
	var sb strings.Builder
	anth, err := dialect.For(ir.DialectAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := anth.WriteStream(&sb, events); err != nil {
		t.Fatal(err)
	}
	out := sb.String()
	if !strings.Contains(out, `"type":"tool_use"`) {
		t.Fatalf("expected tool_use block: %s", out)
	}
	if !strings.Contains(out, `"name":"get_weather"`) {
		t.Fatalf("expected tool name: %s", out)
	}
	if !strings.Contains(out, "input_json_delta") {
		t.Fatalf("expected input_json_delta: %s", out)
	}
	if !strings.Contains(out, `"stop_reason":"tool_use"`) {
		t.Fatalf("expected tool_use stop_reason: %s", out)
	}
}

func TestStreamToolCallsAnthropicToOpenAI(t *testing.T) {
	sse := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"m1","role":"assistant","model":"claude"}}`,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"call_1","name":"get_weather","input":{}}}`,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"city\":\"SF\"}"}}`,
		`event: content_block_stop`,
		`data: {"type":"content_block_stop","index":0}`,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"input_tokens":5,"output_tokens":3}}`,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")

	events := dialect.ParseAnthropicSSE(strings.NewReader(sse))
	var sb strings.Builder
	oa, err := dialect.For(ir.DialectOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := oa.WriteStream(&sb, events); err != nil {
		t.Fatal(err)
	}
	out := sb.String()
	if !strings.Contains(out, `"tool_calls"`) {
		t.Fatalf("expected tool_calls in openai stream: %s", out)
	}
	if !strings.Contains(out, `"name":"get_weather"`) {
		t.Fatalf("expected tool name: %s", out)
	}
	if !strings.Contains(out, "get_weather") || !strings.Contains(out, `SF`) {
		t.Fatalf("expected argument fragments: %s", out)
	}
	if !strings.Contains(out, `"finish_reason":"tool_calls"`) {
		t.Fatalf("expected finish_reason tool_calls: %s", out)
	}
}
