package grpcprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	extensionv1 "github.com/curefatih/afi/gen/proto/afi/extension/v1"
	"github.com/curefatih/afi/sdk/chatir"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
)

func (a *irCapableProvider) ChatIR(ctx context.Context, cfg sdkprovider.ProviderConfig, targetModel string, req chatir.Request) (chatir.Result, error) {
	return a.ProviderAdapter.chatIR(ctx, cfg, targetModel, req)
}

func (a *ProviderAdapter) chatIR(ctx context.Context, cfg sdkprovider.ProviderConfig, targetModel string, req chatir.Request) (chatir.Result, error) {
	if a == nil || a.irClient == nil {
		return chatir.Result{}, fmt.Errorf("grpc provider %q: typed ChatIR is not available", a.typ)
	}
	cctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	pbReq := &extensionv1.ChatIRRequest{
		Config:      providerConfigProto(cfg),
		TargetModel: targetModel,
		Request:     chatIRRequestProto(req),
	}
	if req.Stream {
		return a.chatIRStream(cctx, pbReq)
	}
	resp, err := a.irClient.ChatIR(cctx, pbReq)
	if err != nil {
		return chatir.Result{}, fmt.Errorf("grpc provider %q ChatIR: %w", a.typ, err)
	}
	return chatIRResultFromProto(resp), nil
}

func (a *ProviderAdapter) chatIRStream(ctx context.Context, pbReq *extensionv1.ChatIRRequest) (chatir.Result, error) {
	stream, err := a.irClient.ChatIRStream(ctx, pbReq)
	if err != nil {
		return chatir.Result{}, fmt.Errorf("grpc provider %q ChatIRStream: %w", a.typ, err)
	}
	out := make(chan chatir.StreamEvent, 16)
	go func() {
		defer close(out)
		for {
			ev, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				out <- chatir.StreamEvent{Kind: chatir.StreamError, Err: err}
				return
			}
			out <- chatIRStreamEventFromProto(ev)
		}
	}()
	return chatir.Result{StatusCode: 200, Events: out}, nil
}

func providerConfigProto(cfg sdkprovider.ProviderConfig) *extensionv1.ProviderConfig {
	return &extensionv1.ProviderConfig{
		Id:        cfg.ID,
		Type:      cfg.Type,
		BaseUrl:   cfg.BaseURL,
		ApiKeyEnv: cfg.APIKeyEnv,
		Name:      cfg.Name,
		Capabilities: &extensionv1.Capabilities{
			Chat:      cfg.Capabilities.Chat,
			Stream:    cfg.Capabilities.Stream,
			Tts:       cfg.Capabilities.TTS,
			Stt:       cfg.Capabilities.STT,
			Embedding: cfg.Capabilities.Embedding,
		},
	}
}

func chatIRRequestProto(req chatir.Request) *extensionv1.ChatIRCompletionRequest {
	out := &extensionv1.ChatIRCompletionRequest{
		Model:     req.Model,
		System:    req.System,
		MaxTokens: int32(req.MaxTokens),
		Stream:    req.Stream,
		Stop:      append([]string(nil), req.Stop...),
	}
	if req.Temperature != nil {
		v := *req.Temperature
		out.Temperature = &v
	}
	if req.TopP != nil {
		v := *req.TopP
		out.TopP = &v
	}
	for _, msg := range req.Messages {
		out.Messages = append(out.Messages, chatIRMessageProto(msg))
	}
	for _, tool := range req.Tools {
		out.Tools = append(out.Tools, &extensionv1.ChatIRTool{
			Name: tool.Name, Description: tool.Description, Parameters: append([]byte(nil), tool.Parameters...),
		})
	}
	if req.ToolChoice != nil {
		out.ToolChoice = &extensionv1.ChatIRToolChoice{
			Mode: string(req.ToolChoice.Mode), Name: req.ToolChoice.Name,
		}
	}
	return out
}

func chatIRMessageProto(msg chatir.Message) *extensionv1.ChatIRMessage {
	out := &extensionv1.ChatIRMessage{
		Role: msg.Role, Content: msg.Content, ToolCallId: msg.ToolCallID,
	}
	for _, part := range msg.Parts {
		p := &extensionv1.ChatIRContentPart{Type: string(part.Type), Text: part.Text}
		if part.Image != nil {
			p.Image = &extensionv1.ChatIRImageSource{
				Url: part.Image.URL, MediaType: part.Image.MediaType, Data: part.Image.Data,
			}
		}
		out.Parts = append(out.Parts, p)
	}
	for _, call := range msg.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, &extensionv1.ChatIRToolCall{
			Id: call.ID, Name: call.Name, Arguments: call.Arguments,
		})
	}
	return out
}

func chatIRResultFromProto(resp *extensionv1.ChatIRResponse) chatir.Result {
	if resp == nil {
		return chatir.Result{StatusCode: 502}
	}
	status := int(resp.GetStatusCode())
	if status == 0 {
		status = 200
	}
	hdr := map[string][]string{}
	for k, v := range resp.GetHeaders() {
		if k == "" {
			continue
		}
		hdr[k] = []string{v}
	}
	result := chatir.Result{StatusCode: status, Header: hdr, ErrorBody: append([]byte(nil), resp.GetErrorBody()...)}
	if c := resp.GetCompletion(); c != nil {
		mapped := chatIRCompletionFromProto(c)
		result.Response = &mapped
	}
	return result
}

func chatIRCompletionFromProto(c *extensionv1.ChatIRCompletion) chatir.Response {
	out := chatir.Response{
		ID: c.GetId(), Model: c.GetModel(), Role: c.GetRole(),
		Content: c.GetContent(), FinishReason: c.GetFinishReason(),
	}
	if u := c.GetUsage(); u != nil {
		out.Usage = chatir.Usage{PromptTokens: u.GetPromptTokens(), CompletionTokens: u.GetCompletionTokens()}
	}
	for _, call := range c.GetToolCalls() {
		out.ToolCalls = append(out.ToolCalls, chatir.ToolCall{
			ID: call.GetId(), Name: call.GetName(), Arguments: call.GetArguments(),
		})
	}
	return out
}

func chatIRStreamEventFromProto(ev *extensionv1.ChatIRStreamEvent) chatir.StreamEvent {
	if ev == nil {
		return chatir.StreamEvent{Kind: chatir.StreamError, Err: errors.New("empty ChatIR stream event")}
	}
	out := chatir.StreamEvent{
		ID: ev.GetId(), Model: ev.GetModel(), Role: ev.GetRole(), Text: ev.GetText(),
		FinishReason: ev.GetFinishReason(),
		ToolIndex:    int(ev.GetToolIndex()), ToolID: ev.GetToolId(), ToolName: ev.GetToolName(),
		ArgsDelta: ev.GetArgsDelta(),
	}
	if u := ev.GetUsage(); u != nil {
		usage := chatir.Usage{PromptTokens: u.GetPromptTokens(), CompletionTokens: u.GetCompletionTokens()}
		out.Usage = &usage
	}
	if errMsg := ev.GetError(); errMsg != "" {
		out.Err = errors.New(errMsg)
	}
	switch ev.GetKind() {
	case extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_MESSAGE_START:
		out.Kind = chatir.StreamMessageStart
	case extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_TEXT_DELTA:
		out.Kind = chatir.StreamTextDelta
	case extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_TOOL_CALL_START:
		out.Kind = chatir.StreamToolCallStart
	case extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_TOOL_CALL_DELTA:
		out.Kind = chatir.StreamToolCallDelta
	case extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_MESSAGE_END:
		out.Kind = chatir.StreamMessageEnd
	case extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_ERROR:
		out.Kind = chatir.StreamError
	default:
		out.Kind = chatir.StreamError
		if out.Err == nil {
			out.Err = fmt.Errorf("unknown ChatIR stream event kind %v", ev.GetKind())
		}
	}
	return out
}

// ChatIRRequestFromProto converts a protobuf ChatIR completion request into chatir.Request.
func ChatIRRequestFromProto(in *extensionv1.ChatIRCompletionRequest) chatir.Request {
	if in == nil {
		return chatir.Request{}
	}
	out := chatir.Request{
		Model: in.GetModel(), System: in.GetSystem(), MaxTokens: int(in.GetMaxTokens()),
		Stream: in.GetStream(), Stop: append([]string(nil), in.GetStop()...),
	}
	if in.Temperature != nil {
		v := in.GetTemperature()
		out.Temperature = &v
	}
	if in.TopP != nil {
		v := in.GetTopP()
		out.TopP = &v
	}
	for _, msg := range in.GetMessages() {
		out.Messages = append(out.Messages, chatIRMessageFromProto(msg))
	}
	for _, tool := range in.GetTools() {
		out.Tools = append(out.Tools, chatir.Tool{
			Name: tool.GetName(), Description: tool.GetDescription(),
			Parameters: json.RawMessage(append([]byte(nil), tool.GetParameters()...)),
		})
	}
	if tc := in.GetToolChoice(); tc != nil {
		out.ToolChoice = &chatir.ToolChoice{Mode: chatir.ToolChoiceMode(tc.GetMode()), Name: tc.GetName()}
	}
	return out
}

func chatIRMessageFromProto(msg *extensionv1.ChatIRMessage) chatir.Message {
	if msg == nil {
		return chatir.Message{}
	}
	out := chatir.Message{Role: msg.GetRole(), Content: msg.GetContent(), ToolCallID: msg.GetToolCallId()}
	for _, part := range msg.GetParts() {
		p := chatir.ContentPart{Type: chatir.ContentPartType(part.GetType()), Text: part.GetText()}
		if img := part.GetImage(); img != nil {
			p.Image = &chatir.ImageSource{URL: img.GetUrl(), MediaType: img.GetMediaType(), Data: img.GetData()}
		}
		out.Parts = append(out.Parts, p)
	}
	for _, call := range msg.GetToolCalls() {
		out.ToolCalls = append(out.ToolCalls, chatir.ToolCall{
			ID: call.GetId(), Name: call.GetName(), Arguments: call.GetArguments(),
		})
	}
	return out
}

func chatIRCompletionProto(resp chatir.Response) *extensionv1.ChatIRCompletion {
	out := &extensionv1.ChatIRCompletion{
		Id: resp.ID, Model: resp.Model, Role: resp.Role, Content: resp.Content,
		FinishReason: resp.FinishReason,
		Usage: &extensionv1.ChatIRUsage{
			PromptTokens: resp.Usage.PromptTokens, CompletionTokens: resp.Usage.CompletionTokens,
		},
	}
	for _, call := range resp.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, &extensionv1.ChatIRToolCall{
			Id: call.ID, Name: call.Name, Arguments: call.Arguments,
		})
	}
	return out
}

// ChatIRResponseProto builds a unary ChatIRResponse from a chatir.Result.
func ChatIRResponseProto(result chatir.Result) *extensionv1.ChatIRResponse {
	hdr := map[string]string{}
	for k, vals := range result.Header {
		if k == "" || len(vals) == 0 {
			continue
		}
		hdr[k] = vals[0]
	}
	out := &extensionv1.ChatIRResponse{
		StatusCode: int32(result.StatusCode),
		Headers:    hdr,
		ErrorBody:  append([]byte(nil), result.ErrorBody...),
	}
	if result.Response != nil {
		out.Completion = chatIRCompletionProto(*result.Response)
	}
	return out
}

// ChatIRStreamEventProto converts a chatir stream event to protobuf.
func ChatIRStreamEventProto(ev chatir.StreamEvent) *extensionv1.ChatIRStreamEvent {
	out := &extensionv1.ChatIRStreamEvent{
		Id: ev.ID, Model: ev.Model, Role: ev.Role, Text: ev.Text,
		FinishReason: ev.FinishReason,
		ToolIndex:    int32(ev.ToolIndex), ToolId: ev.ToolID, ToolName: ev.ToolName,
		ArgsDelta: ev.ArgsDelta,
	}
	if ev.Usage != nil {
		out.Usage = &extensionv1.ChatIRUsage{
			PromptTokens: ev.Usage.PromptTokens, CompletionTokens: ev.Usage.CompletionTokens,
		}
	}
	if ev.Err != nil {
		out.Error = ev.Err.Error()
	}
	switch ev.Kind {
	case chatir.StreamMessageStart:
		out.Kind = extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_MESSAGE_START
	case chatir.StreamTextDelta:
		out.Kind = extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_TEXT_DELTA
	case chatir.StreamToolCallStart:
		out.Kind = extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_TOOL_CALL_START
	case chatir.StreamToolCallDelta:
		out.Kind = extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_TOOL_CALL_DELTA
	case chatir.StreamMessageEnd:
		out.Kind = extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_MESSAGE_END
	case chatir.StreamError:
		out.Kind = extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_ERROR
	default:
		out.Kind = extensionv1.ChatIRStreamEventKind_CHAT_IR_STREAM_EVENT_KIND_UNSPECIFIED
	}
	return out
}
