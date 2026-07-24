package grpcprovider

import (
	"context"
	"fmt"
	"time"

	extensionv1 "github.com/curefatih/afi/gen/proto/afi/extension/v1"
	sdkhook "github.com/curefatih/afi/sdk/hook"
	"google.golang.org/protobuf/types/known/structpb"
)

// BeforeCallAdapter implements sdk/hook.BeforeCallHook over gRPC.
type BeforeCallAdapter struct {
	client  extensionv1.HookClient
	name    string
	timeout time.Duration
}

func (a *BeforeCallAdapter) Name() string {
	if a == nil || a.name == "" {
		return "grpc"
	}
	return a.name
}

func (a *BeforeCallAdapter) BeforeCall(ctx context.Context, call *sdkhook.CallContext) (sdkhook.CallDecision, error) {
	if a == nil || a.client == nil || call == nil {
		return sdkhook.Allow(), nil
	}
	cctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	resp, err := a.client.BeforeCall(cctx, &extensionv1.BeforeCallRequest{Call: callContextToProto(call)})
	if err != nil {
		return sdkhook.CallDecision{}, fmt.Errorf("grpc hook %s BeforeCall: %w", a.Name(), err)
	}
	if resp.GetCall() != nil {
		applyCallContext(call, resp.GetCall())
	}
	return decisionFromProto(resp.GetDecision()), nil
}

// AfterCallAdapter implements sdk/hook.AfterCallHook over gRPC.
type AfterCallAdapter struct {
	client  extensionv1.HookClient
	name    string
	timeout time.Duration
}

func (a *AfterCallAdapter) Name() string {
	if a == nil || a.name == "" {
		return "grpc"
	}
	return a.name
}

func (a *AfterCallAdapter) AfterCall(ctx context.Context, call *sdkhook.CallContext, info sdkhook.AfterCallInfo) error {
	if a == nil || a.client == nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	_, err := a.client.AfterCall(cctx, &extensionv1.AfterCallRequest{
		Call: callContextToProto(call),
		Info: &extensionv1.AfterCallInfo{
			Status:           info.Status,
			LatencyMs:        info.LatencyMs,
			ProviderType:     info.ProviderType,
			TargetModel:      info.TargetModel,
			PromptTokens:     info.PromptTokens,
			CompletionTokens: info.CompletionTokens,
		},
	})
	if err != nil {
		return fmt.Errorf("grpc hook %s AfterCall: %w", a.Name(), err)
	}
	return nil
}

// BeforeChatAdapter implements sdk/hook.ChatHook over gRPC.
type BeforeChatAdapter struct {
	client  extensionv1.HookClient
	name    string
	timeout time.Duration
}

func (a *BeforeChatAdapter) Name() string {
	if a == nil || a.name == "" {
		return "grpc:before_chat"
	}
	return a.name
}

func (a *BeforeChatAdapter) BeforeChat(ctx context.Context, body []byte) ([]byte, error) {
	if a == nil || a.client == nil {
		return body, nil
	}
	cctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	resp, err := a.client.BeforeChat(cctx, &extensionv1.BeforeChatRequest{Body: body})
	if err != nil {
		return nil, fmt.Errorf("grpc hook %s BeforeChat: %w", a.Name(), err)
	}
	if resp == nil || resp.Body == nil {
		return body, nil
	}
	return resp.Body, nil
}

// AfterChatAdapter implements sdk/hook.AfterChatHook over gRPC.
type AfterChatAdapter struct {
	client  extensionv1.HookClient
	name    string
	timeout time.Duration
}

func (a *AfterChatAdapter) Name() string {
	if a == nil || a.name == "" {
		return "grpc:after_chat"
	}
	return a.name
}

func (a *AfterChatAdapter) AfterChat(ctx context.Context, info sdkhook.AfterChatInfo) error {
	if a == nil || a.client == nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	_, err := a.client.AfterChat(cctx, &extensionv1.AfterChatRequest{
		Info: &extensionv1.AfterChatInfo{
			Model:        info.Model,
			Status:       info.Status,
			LatencyMs:    info.LatencyMs,
			ProviderType: info.ProviderType,
			TargetModel:  info.TargetModel,
		},
	})
	if err != nil {
		return fmt.Errorf("grpc hook %s AfterChat: %w", a.Name(), err)
	}
	return nil
}

func callContextToProto(call *sdkhook.CallContext) *extensionv1.CallContext {
	if call == nil {
		return nil
	}
	meta, _ := structpb.NewStruct(normalizeMetadata(call.Metadata))
	return &extensionv1.CallContext{
		Principal: &extensionv1.Principal{
			OrganizationId: call.Principal.OrganizationID,
			ProjectId:      call.Principal.ProjectID,
			ApiKeyId:       call.Principal.APIKeyID,
			Kind:           call.Principal.Kind,
			OwnerUserId:    call.Principal.OwnerUserID,
			Name:           call.Principal.Name,
		},
		Route: &extensionv1.RouteContext{
			Model:    call.Route.Model,
			Path:     call.Route.Path,
			Stream:   call.Route.Stream,
			Modality: call.Route.Modality,
		},
		Tags:            copyStringMap(call.Tags),
		Headers:         copyStringMap(call.Headers),
		Metadata:        meta,
		Body:            call.Body,
		RequestHeaders:  copyStringMap(call.RequestHeaders),
		ResponseHeaders: copyStringMap(call.ResponseHeaders),
	}
}

func applyCallContext(dst *sdkhook.CallContext, src *extensionv1.CallContext) {
	if dst == nil || src == nil {
		return
	}
	if src.Principal != nil {
		dst.Principal = sdkhook.Principal{
			OrganizationID: src.Principal.OrganizationId,
			ProjectID:      src.Principal.ProjectId,
			APIKeyID:       src.Principal.ApiKeyId,
			Kind:           src.Principal.Kind,
			OwnerUserID:    src.Principal.OwnerUserId,
			Name:           src.Principal.Name,
		}
	}
	if src.Route != nil {
		dst.Route = sdkhook.RouteContext{
			Model:    src.Route.Model,
			Path:     src.Route.Path,
			Stream:   src.Route.Stream,
			Modality: src.Route.Modality,
		}
	}
	if src.Tags != nil {
		dst.Tags = copyStringMap(src.Tags)
	}
	if src.Headers != nil {
		dst.Headers = copyStringMap(src.Headers)
	}
	if src.Metadata != nil {
		dst.Metadata = src.Metadata.AsMap()
	}
	if src.Body != nil {
		dst.Body = append([]byte(nil), src.Body...)
	}
	if src.RequestHeaders != nil {
		dst.RequestHeaders = copyStringMap(src.RequestHeaders)
	}
	if src.ResponseHeaders != nil {
		dst.ResponseHeaders = copyStringMap(src.ResponseHeaders)
	}
}

func decisionFromProto(d *extensionv1.CallDecision) sdkhook.CallDecision {
	if d == nil {
		return sdkhook.Allow()
	}
	return sdkhook.CallDecision{
		Allow:   d.Allow,
		Status:  int(d.Status),
		Reason:  d.Reason,
		Message: d.Message,
		Headers: copyStringMap(d.Headers),
	}
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// normalizeMetadata coerces values to types structpb accepts.
func normalizeMetadata(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		switch v.(type) {
		case nil, bool, float64, string, []any, map[string]any:
			out[k] = v
		case int:
			out[k] = float64(v.(int))
		case int64:
			out[k] = float64(v.(int64))
		case float32:
			out[k] = float64(v.(float32))
		default:
			out[k] = fmt.Sprint(v)
		}
	}
	return out
}
