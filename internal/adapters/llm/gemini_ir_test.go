package llm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/ir"
)

func TestIRToGeminiMapsToolsAndVision(t *testing.T) {
	req := ir.ChatRequest{
		Model:  "route",
		System: "be brief",
		Messages: []ir.Message{
			{
				Role: "user",
				Parts: []ir.ContentPart{
					{Type: ir.ContentText, Text: "inspect"},
					{Type: ir.ContentImage, Image: &ir.ImageSource{
						MediaType: "image/png", Data: "aW1hZ2U=",
					}},
				},
			},
		},
		Tools: []ir.Tool{{
			Name: "inspect", Parameters: json.RawMessage(`{"type":"object"}`),
		}},
		ToolChoice: &ir.ToolChoice{Mode: ir.ToolChoiceTool, Name: "inspect"},
	}
	body, err := irToGemini(req)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		`"inlineData"`, `"functionDeclarations"`, `"allowedFunctionNames":["inspect"]`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s in %s", want, text)
		}
	}
}

func TestGeminiJSONToIRMapsFunctionCall(t *testing.T) {
	resp, err := geminiJSONToIR([]byte(`{
		"candidates":[{
			"content":{"role":"model","parts":[
				{"functionCall":{"id":"call_1","name":"weather","args":{"city":"SF"}}}
			]},
			"finishReason":"STOP"
		}],
		"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":1}
	}`), "gemini-route")
	if err != nil {
		t.Fatal(err)
	}
	if resp.FinishReason != "tool_calls" || len(resp.ToolCalls) != 1 {
		t.Fatalf("response=%+v", resp)
	}
	if resp.ToolCalls[0].ID != "call_1" || resp.ToolCalls[0].Name != "weather" ||
		!strings.Contains(resp.ToolCalls[0].Arguments, "SF") {
		t.Fatalf("tool call=%+v", resp.ToolCalls[0])
	}
}
