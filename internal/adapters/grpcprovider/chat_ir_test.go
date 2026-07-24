package grpcprovider_test

import (
	"encoding/json"
	"errors"
	"testing"

	extensionv1 "github.com/curefatih/afi/gen/proto/afi/extension/v1"
	"github.com/curefatih/afi/internal/adapters/grpcprovider"
	"github.com/curefatih/afi/sdk/chatir"
)

func TestChatIRProtoRoundTrip(t *testing.T) {
	temp := 0.2
	topP := 0.9
	req := chatir.Request{
		Model: "demo", System: "sys", MaxTokens: 64, Temperature: &temp, TopP: &topP,
		Stream: false, Stop: []string{"END"},
		Messages: []chatir.Message{{
			Role: "user", Content: "hi",
			Parts: []chatir.ContentPart{
				{Type: chatir.ContentText, Text: "hi"},
				{Type: chatir.ContentImage, Image: &chatir.ImageSource{URL: "https://x", MediaType: "image/png"}},
			},
		}},
		Tools: []chatir.Tool{{
			Name: "lookup", Description: "d", Parameters: json.RawMessage(`{"type":"object"}`),
		}},
		ToolChoice: &chatir.ToolChoice{Mode: chatir.ToolChoiceTool, Name: "lookup"},
	}
	pb := grpcprovider.ChatIRRequestFromProto(
		mustChatIRRequestProto(t, req),
	)
	if pb.Model != "demo" || pb.System != "sys" || pb.MaxTokens != 64 ||
		pb.Temperature == nil || *pb.Temperature != 0.2 ||
		len(pb.Messages) != 1 || len(pb.Messages[0].Parts) != 2 ||
		len(pb.Tools) != 1 || pb.ToolChoice == nil || pb.ToolChoice.Name != "lookup" {
		t.Fatalf("round trip request=%+v", pb)
	}

	result := chatir.Result{
		StatusCode: 200,
		Header:     map[string][]string{"X-A": {"1"}},
		Response: &chatir.Response{
			ID: "id", Model: "demo", Role: "assistant", Content: "ok",
			FinishReason: "stop",
			ToolCalls:    []chatir.ToolCall{{ID: "c1", Name: "lookup", Arguments: `{}`}},
			Usage:        chatir.Usage{PromptTokens: 2, CompletionTokens: 3},
		},
	}
	mapped := grpcprovider.ChatIRResponseProto(result)
	if mapped.GetStatusCode() != 200 || mapped.GetHeaders()["X-A"] != "1" ||
		mapped.GetCompletion().GetContent() != "ok" ||
		len(mapped.GetCompletion().GetToolCalls()) != 1 {
		t.Fatalf("response=%v", mapped)
	}

	ev := grpcprovider.ChatIRStreamEventProto(chatir.StreamEvent{
		Kind: chatir.StreamError, Err: errors.New("boom"), Text: "x",
	})
	if ev.GetKind() != extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_ERROR ||
		ev.GetError() != "boom" {
		t.Fatalf("event=%v", ev)
	}
}

func mustChatIRRequestProto(t *testing.T, req chatir.Request) *extensionv1.ChatIRCompletionRequest {
	t.Helper()
	// Round-trip through the exported from-proto helper by first building via ChatIRResponse-style fields.
	// Use a tiny local encode by calling ChatIR on a fake is awkward; build with FromProto inverse via public API.
	temp := req.Temperature
	topP := req.TopP
	out := &extensionv1.ChatIRCompletionRequest{
		Model: req.Model, System: req.System, MaxTokens: int32(req.MaxTokens),
		Stream: req.Stream, Stop: append([]string(nil), req.Stop...),
		Temperature: temp, TopP: topP,
	}
	for _, msg := range req.Messages {
		m := &extensionv1.ChatIRMessage{Role: msg.Role, Content: msg.Content, ToolCallId: msg.ToolCallID}
		for _, part := range msg.Parts {
			p := &extensionv1.ChatIRContentPart{Type: string(part.Type), Text: part.Text}
			if part.Image != nil {
				p.Image = &extensionv1.ChatIRImageSource{
					Url: part.Image.URL, MediaType: part.Image.MediaType, Data: part.Image.Data,
				}
			}
			m.Parts = append(m.Parts, p)
		}
		out.Messages = append(out.Messages, m)
	}
	for _, tool := range req.Tools {
		out.Tools = append(out.Tools, &extensionv1.ChatIRTool{
			Name: tool.Name, Description: tool.Description, Parameters: append([]byte(nil), tool.Parameters...),
		})
	}
	if req.ToolChoice != nil {
		out.ToolChoice = &extensionv1.ChatIRToolChoice{Mode: string(req.ToolChoice.Mode), Name: req.ToolChoice.Name}
	}
	return out
}
