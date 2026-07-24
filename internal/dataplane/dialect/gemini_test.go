package dialect_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/dialect"
	"github.com/curefatih/afi/internal/dataplane/ir"
)

func TestGeminiDecodeToolsVisionAndToolResult(t *testing.T) {
	body := []byte(`{
		"systemInstruction":{"parts":[{"text":"be brief"}]},
		"contents":[
			{"role":"user","parts":[
				{"text":"describe and inspect"},
				{"inlineData":{"mimeType":"image/png","data":"aW1hZ2U="}}
			]},
			{"role":"model","parts":[
				{"functionCall":{"id":"call_1","name":"inspect","args":{"kind":"weather"}}}
			]},
			{"role":"user","parts":[
				{"functionResponse":{"name":"inspect","response":{"result":"sunny"}}}
			]}
		],
		"tools":[{"functionDeclarations":[
			{"name":"inspect","description":"inspect data","parameters":{"type":"object"}}
		]}],
		"toolConfig":{"functionCallingConfig":{"mode":"ANY","allowedFunctionNames":["inspect"]}},
		"generationConfig":{"maxOutputTokens":128,"temperature":0.2,"topP":0.9,"stopSequences":["END"]}
	}`)
	codec, err := dialect.For(ir.DialectGemini)
	if err != nil {
		t.Fatal(err)
	}
	req, err := codec.DecodeRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	if req.System != "be brief" || req.MaxTokens != 128 || req.Temperature == nil || req.TopP == nil {
		t.Fatalf("request options=%+v", req)
	}
	if len(req.Tools) != 1 || req.Tools[0].Name != "inspect" {
		t.Fatalf("tools=%+v", req.Tools)
	}
	if req.ToolChoice == nil || req.ToolChoice.Mode != ir.ToolChoiceTool || req.ToolChoice.Name != "inspect" {
		t.Fatalf("tool choice=%+v", req.ToolChoice)
	}
	if len(req.Messages) != 3 {
		t.Fatalf("messages=%+v", req.Messages)
	}
	if len(req.Messages[0].Parts) != 2 || req.Messages[0].Parts[1].Image.Data != "aW1hZ2U=" {
		t.Fatalf("vision parts=%+v", req.Messages[0].Parts)
	}
	if len(req.Messages[1].ToolCalls) != 1 || req.Messages[1].ToolCalls[0].ID != "call_1" {
		t.Fatalf("tool calls=%+v", req.Messages[1])
	}
	if req.Messages[2].Role != "tool" || req.Messages[2].ToolCallID != "call_1" ||
		!strings.Contains(req.Messages[2].Content, "sunny") {
		t.Fatalf("tool result=%+v", req.Messages[2])
	}
}

func TestGeminiEncodeToolCallResponse(t *testing.T) {
	codec, err := dialect.For(ir.DialectGemini)
	if err != nil {
		t.Fatal(err)
	}
	body, err := codec.EncodeResponse(ir.ChatResponse{
		ID: "resp_1", Model: "route-model", Role: "assistant",
		ToolCalls: []ir.ToolCall{{
			ID: "call_1", Name: "weather", Arguments: `{"city":"SF"}`,
		}},
		FinishReason: "tool_calls",
		Usage:        ir.Usage{PromptTokens: 3, CompletionTokens: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	candidates := out["candidates"].([]any)
	candidate := candidates[0].(map[string]any)
	if candidate["finishReason"] != "STOP" {
		t.Fatalf("finish=%v", candidate["finishReason"])
	}
	content := candidate["content"].(map[string]any)
	parts := content["parts"].([]any)
	call := parts[0].(map[string]any)["functionCall"].(map[string]any)
	if call["name"] != "weather" || call["id"] != "call_1" {
		t.Fatalf("call=%v", call)
	}
	args := call["args"].(map[string]any)
	if args["city"] != "SF" {
		t.Fatalf("args=%v", args)
	}
}

func TestGeminiWriteToolCallStream(t *testing.T) {
	codec, err := dialect.For(ir.DialectGemini)
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan ir.StreamEvent, 5)
	events <- ir.StreamEvent{Kind: ir.StreamMessageStart, Role: "assistant"}
	events <- ir.StreamEvent{
		Kind: ir.StreamToolCallStart, ToolIndex: 0, ToolID: "call_1", ToolName: "weather",
	}
	events <- ir.StreamEvent{Kind: ir.StreamToolCallDelta, ToolIndex: 0, ArgsDelta: `{"city":"SF"}`}
	events <- ir.StreamEvent{
		Kind: ir.StreamMessageEnd, FinishReason: "tool_calls",
		Usage: &ir.Usage{PromptTokens: 4, CompletionTokens: 2},
	}
	close(events)

	var out strings.Builder
	prompt, completion, err := codec.WriteStream(&out, events)
	if err != nil {
		t.Fatal(err)
	}
	if prompt != 4 || completion != 2 {
		t.Fatalf("usage=%d/%d", prompt, completion)
	}
	if !strings.Contains(out.String(), `"functionCall"`) ||
		!strings.Contains(out.String(), `"name":"weather"`) ||
		!strings.Contains(out.String(), `"finishReason":"STOP"`) {
		t.Fatalf("stream=%s", out.String())
	}
}

func TestGeminiRejectsMultipleCandidates(t *testing.T) {
	codec, _ := dialect.For(ir.DialectGemini)
	_, err := codec.DecodeRequest([]byte(`{
		"contents":[{"role":"user","parts":[{"text":"hi"}]}],
		"generationConfig":{"candidateCount":2}
	}`))
	if err == nil {
		t.Fatal("expected candidateCount rejection")
	}
	if _, ok := ir.AsUnsupported(err); !ok {
		t.Fatalf("err=%v", err)
	}
}

func TestGeminiRejectsUnsupportedFields(t *testing.T) {
	codec, _ := dialect.For(ir.DialectGemini)
	cases := []struct {
		name string
		body string
		feat string
	}{
		{
			name: "cachedContent",
			body: `{"contents":[{"role":"user","parts":[{"text":"hi"}]}],"cachedContent":"cached/xyz"}`,
			feat: "cached_content",
		},
		{
			name: "safetySettings",
			body: `{"contents":[{"role":"user","parts":[{"text":"hi"}]}],"safetySettings":[{"category":"HARM_CATEGORY_HATE_SPEECH","threshold":"BLOCK_NONE"}]}`,
			feat: "safety_settings",
		},
		{
			name: "responseMimeType",
			body: `{"contents":[{"role":"user","parts":[{"text":"hi"}]}],"generationConfig":{"responseMimeType":"application/json"}}`,
			feat: "responseMimeType",
		},
		{
			name: "thinkingConfig",
			body: `{"contents":[{"role":"user","parts":[{"text":"hi"}]}],"generationConfig":{"thinkingConfig":{"thinkingBudget":1024}}}`,
			feat: "thinkingConfig",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := codec.DecodeRequest([]byte(tc.body))
			if err == nil {
				t.Fatal("expected unsupported rejection")
			}
			u, ok := ir.AsUnsupported(err)
			if !ok {
				t.Fatalf("err=%v", err)
			}
			if u.Feature != tc.feat {
				t.Fatalf("feature=%q want=%q", u.Feature, tc.feat)
			}
		})
	}
}
