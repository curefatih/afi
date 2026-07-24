package llm

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/curefatih/afi/internal/dataplane/ir"
)

func TestIRToConverseInputTextAndSystem(t *testing.T) {
	t.Parallel()
	temp := 0.5
	req := ir.ChatRequest{
		System:      "be helpful",
		MaxTokens:   64,
		Temperature: &temp,
		Messages: []ir.Message{
			{Role: "user", Content: "hello"},
		},
	}
	in, err := irToConverseInput("anthropic.claude-3-haiku", req)
	if err != nil {
		t.Fatal(err)
	}
	if aws.ToString(in.ModelId) != "anthropic.claude-3-haiku" {
		t.Fatalf("model=%q", aws.ToString(in.ModelId))
	}
	if len(in.System) != 1 {
		t.Fatalf("system=%v", in.System)
	}
	sys, ok := in.System[0].(*types.SystemContentBlockMemberText)
	if !ok || sys.Value != "be helpful" {
		t.Fatalf("system=%v", in.System[0])
	}
	if len(in.Messages) != 1 || in.Messages[0].Role != types.ConversationRoleUser {
		t.Fatalf("messages=%v", in.Messages)
	}
	text, ok := in.Messages[0].Content[0].(*types.ContentBlockMemberText)
	if !ok || text.Value != "hello" {
		t.Fatalf("content=%v", in.Messages[0].Content)
	}
	if in.InferenceConfig == nil || aws.ToInt32(in.InferenceConfig.MaxTokens) != 64 {
		t.Fatalf("inference=%v", in.InferenceConfig)
	}
}

func TestIRToConverseInputToolsAndVision(t *testing.T) {
	t.Parallel()
	req := ir.ChatRequest{
		Messages: []ir.Message{{
			Role: "user",
			Parts: []ir.ContentPart{
				{Type: ir.ContentText, Text: "what is this?"},
				{Type: ir.ContentImage, Image: &ir.ImageSource{
					MediaType: "image/png", Data: "aGVsbG8=",
				}},
			},
		}},
		Tools: []ir.Tool{{
			Name:        "lookup",
			Description: "look up",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`),
		}},
		ToolChoice: &ir.ToolChoice{Mode: ir.ToolChoiceAuto},
	}
	in, err := irToConverseInput("model", req)
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Messages[0].Content) != 2 {
		t.Fatalf("content len=%d", len(in.Messages[0].Content))
	}
	if _, ok := in.Messages[0].Content[1].(*types.ContentBlockMemberImage); !ok {
		t.Fatalf("expected image block, got %T", in.Messages[0].Content[1])
	}
	if in.ToolConfig == nil || len(in.ToolConfig.Tools) != 1 {
		t.Fatalf("tools=%v", in.ToolConfig)
	}
	if _, ok := in.ToolConfig.ToolChoice.(*types.ToolChoiceMemberAuto); !ok {
		t.Fatalf("toolChoice=%T", in.ToolConfig.ToolChoice)
	}
}

func TestIRToConverseInputURLImageUnsupported(t *testing.T) {
	t.Parallel()
	req := ir.ChatRequest{
		Messages: []ir.Message{{
			Role: "user",
			Parts: []ir.ContentPart{{
				Type: ir.ContentImage, Image: &ir.ImageSource{URL: "https://x/y.png"},
			}},
		}},
	}
	_, err := irToConverseInput("model", req)
	if _, ok := ir.AsUnsupported(err); !ok {
		t.Fatalf("want UnsupportedError, got %v", err)
	}
}

func TestConverseOutputToIR(t *testing.T) {
	t.Parallel()
	out := &bedrockruntime.ConverseOutput{
		StopReason: types.StopReasonEndTurn,
		Usage: &types.TokenUsage{
			InputTokens:  aws.Int32(3),
			OutputTokens: aws.Int32(5),
			TotalTokens:  aws.Int32(8),
		},
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{
				Role: types.ConversationRoleAssistant,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: "hi there"},
				},
			},
		},
	}
	resp, err := converseOutputToIR(out, "m")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hi there" || resp.FinishReason != "stop" {
		t.Fatalf("resp=%+v", resp)
	}
	if resp.Usage.PromptTokens != 3 || resp.Usage.CompletionTokens != 5 {
		t.Fatalf("usage=%+v", resp.Usage)
	}
}

func TestConverseOutputToIRToolUse(t *testing.T) {
	t.Parallel()
	out := &bedrockruntime.ConverseOutput{
		StopReason: types.StopReasonToolUse,
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{
				Role: types.ConversationRoleAssistant,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberToolUse{
						Value: types.ToolUseBlock{
							ToolUseId: aws.String("call_1"),
							Name:      aws.String("lookup"),
							Input:     document.NewLazyDocument(map[string]any{"q": "x"}),
						},
					},
				},
			},
		},
	}
	resp, err := converseOutputToIR(out, "m")
	if err != nil {
		t.Fatal(err)
	}
	if resp.FinishReason != "tool_calls" || len(resp.ToolCalls) != 1 {
		t.Fatalf("resp=%+v", resp)
	}
	if resp.ToolCalls[0].Name != "lookup" || resp.ToolCalls[0].ID != "call_1" {
		t.Fatalf("call=%+v", resp.ToolCalls[0])
	}
	// Arguments may be "{}" when Input is a request-side LazyDocument stub;
	// live Bedrock responses provide an unmarshalable document interface.
}

func TestMapBedrockStopReason(t *testing.T) {
	t.Parallel()
	cases := map[types.StopReason]string{
		types.StopReasonEndTurn:      "stop",
		types.StopReasonMaxTokens:    "length",
		types.StopReasonToolUse:      "tool_calls",
		types.StopReasonStopSequence: "stop",
	}
	for in, want := range cases {
		if got := mapBedrockStopReason(in); got != want {
			t.Fatalf("%q => %q want %q", in, got, want)
		}
	}
}

func TestMapConverseStreamEvents(t *testing.T) {
	t.Parallel()
	next := 0
	indexes := map[int]int{}
	start := mapConverseStreamEvent(
		&types.ConverseStreamOutputMemberMessageStart{
			Value: types.MessageStartEvent{Role: types.ConversationRoleAssistant},
		},
		"m", "id", &next, indexes,
	)
	if len(start) != 1 || start[0].Kind != ir.StreamMessageStart {
		t.Fatalf("start=%v", start)
	}
	delta := mapConverseStreamEvent(
		&types.ConverseStreamOutputMemberContentBlockDelta{
			Value: types.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta:             &types.ContentBlockDeltaMemberText{Value: "hi"},
			},
		},
		"m", "id", &next, indexes,
	)
	if len(delta) != 1 || delta[0].Text != "hi" {
		t.Fatalf("delta=%v", delta)
	}
}

func TestRegionFromBaseURL(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"https://bedrock-runtime.us-west-2.amazonaws.com":    "us-west-2",
		"https://bedrock-runtime.us-east-1.amazonaws.com/":   "us-east-1",
		"https://bedrock-runtime.eu-central-1.amazonaws.com": "eu-central-1",
		"https://example.com":                                "",
		"":                                                   "",
	}
	for in, want := range cases {
		if got := regionFromBaseURL(in); got != want {
			t.Fatalf("%q => %q want %q", in, got, want)
		}
	}
}

func TestParseAWSStaticCreds(t *testing.T) {
	t.Parallel()
	ak, sk, st, ok := parseAWSStaticCreds("AKIAxxx:secret/value")
	if !ok || ak != "AKIAxxx" || sk != "secret/value" || st != "" {
		t.Fatalf("got %q %q %q %v", ak, sk, st, ok)
	}
	ak, sk, st, ok = parseAWSStaticCreds("AKIAxxx:secret:session-token")
	if !ok || ak != "AKIAxxx" || sk != "secret" || st != "session-token" {
		t.Fatalf("got %q %q %q %v", ak, sk, st, ok)
	}
	if _, _, _, ok := parseAWSStaticCreds(""); ok {
		t.Fatal("empty should be false")
	}
	if _, _, _, ok := parseAWSStaticCreds("nocolon"); ok {
		t.Fatal("malformed should be false")
	}
}
