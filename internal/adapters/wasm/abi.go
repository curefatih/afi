package wasm

import (
	"encoding/base64"
	"encoding/json"

	sdkhook "github.com/curefatih/afi/sdk/hook"
)

// Wire types for the host↔guest JSON ABI (see docs/hooks/wasm.md).

type principalWire struct {
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	APIKeyID       string `json:"api_key_id"`
	Kind           string `json:"kind"`
	OwnerUserID    string `json:"owner_user_id"`
	Name           string `json:"name"`
}

type routeWire struct {
	Model    string `json:"model"`
	Path     string `json:"path"`
	Stream   bool   `json:"stream"`
	Modality string `json:"modality"`
}

type beforeCallIn struct {
	Principal       principalWire     `json:"principal"`
	Route           routeWire         `json:"route"`
	Tags            map[string]string `json:"tags"`
	Metadata        map[string]any    `json:"metadata"`
	BodyB64         string            `json:"body_b64"`
	Config          json.RawMessage   `json:"config,omitempty"`
	RequestHeaders  map[string]string `json:"request_headers"`
	ResponseHeaders map[string]string `json:"response_headers"`
}

type beforeCallOut struct {
	Allow           bool              `json:"allow"`
	Status          int               `json:"status"`
	Reason          string            `json:"reason"`
	Message         string            `json:"message"`
	Headers         map[string]string `json:"headers"` // deny response headers (legacy + CallDecision.Headers)
	Tags            map[string]string `json:"tags"`
	Metadata        map[string]any    `json:"metadata"`
	BodyB64         *string           `json:"body_b64"`
	RequestHeaders  map[string]string `json:"request_headers"`
	ResponseHeaders map[string]string `json:"response_headers"`
}

type beforeChatIn struct {
	BodyB64 string          `json:"body_b64"`
	Config  json.RawMessage `json:"config,omitempty"`
}

type beforeChatOut struct {
	BodyB64 string `json:"body_b64"`
}

type afterCallIn struct {
	Principal principalWire     `json:"principal"`
	Route     routeWire         `json:"route"`
	Tags      map[string]string `json:"tags"`
	Metadata  map[string]any    `json:"metadata"`
	Status    string            `json:"status"`
	LatencyMs int64             `json:"latency_ms"`
	Provider  string            `json:"provider_type"`
	Target    string            `json:"target_model"`
	PromptTok int64             `json:"prompt_tokens"`
	ComplTok  int64             `json:"completion_tokens"`
	Config    json.RawMessage   `json:"config,omitempty"`
}

func encodeBeforeCallIn(call *sdkhook.CallContext, config json.RawMessage) ([]byte, error) {
	in := beforeCallIn{
		Principal: principalWire{
			OrganizationID: call.Principal.OrganizationID,
			ProjectID:      call.Principal.ProjectID,
			APIKeyID:       call.Principal.APIKeyID,
			Kind:           call.Principal.Kind,
			OwnerUserID:    call.Principal.OwnerUserID,
			Name:           call.Principal.Name,
		},
		Route: routeWire{
			Model:    call.Route.Model,
			Path:     call.Route.Path,
			Stream:   call.Route.Stream,
			Modality: call.Route.Modality,
		},
		Tags:            call.Tags,
		Metadata:        call.Metadata,
		Config:          config,
		RequestHeaders:  call.RequestHeaders,
		ResponseHeaders: call.ResponseHeaders,
	}
	if in.Tags == nil {
		in.Tags = map[string]string{}
	}
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	if in.RequestHeaders == nil {
		in.RequestHeaders = map[string]string{}
	}
	if in.ResponseHeaders == nil {
		in.ResponseHeaders = map[string]string{}
	}
	if len(call.Body) > 0 {
		in.BodyB64 = base64.StdEncoding.EncodeToString(call.Body)
	}
	return json.Marshal(in)
}

func applyBeforeCallOut(call *sdkhook.CallContext, raw []byte) (sdkhook.CallDecision, error) {
	var out beforeCallOut
	if err := json.Unmarshal(raw, &out); err != nil {
		return sdkhook.CallDecision{}, err
	}
	if out.Tags != nil {
		call.Tags = out.Tags
	}
	if out.Metadata != nil {
		call.Metadata = out.Metadata
	}
	if out.RequestHeaders != nil {
		call.RequestHeaders = out.RequestHeaders
	}
	if out.ResponseHeaders != nil {
		call.ResponseHeaders = out.ResponseHeaders
	}
	if out.BodyB64 != nil {
		if *out.BodyB64 == "" {
			call.Body = nil
		} else {
			b, err := base64.StdEncoding.DecodeString(*out.BodyB64)
			if err != nil {
				return sdkhook.CallDecision{}, err
			}
			call.Body = b
		}
	}
	d := sdkhook.CallDecision{
		Allow:   out.Allow,
		Status:  out.Status,
		Reason:  out.Reason,
		Message: out.Message,
		Headers: out.Headers,
	}
	if !d.Allow && d.Status == 0 {
		d.Status = 403
	}
	// Merge ResponseHeaders into deny Headers when denying.
	if !d.Allow && len(call.ResponseHeaders) > 0 {
		if d.Headers == nil {
			d.Headers = map[string]string{}
		}
		for k, v := range call.ResponseHeaders {
			if _, ok := d.Headers[k]; !ok {
				d.Headers[k] = v
			}
		}
	}
	return d, nil
}

func encodeBeforeChatIn(body []byte, config json.RawMessage) ([]byte, error) {
	in := beforeChatIn{Config: config}
	if len(body) > 0 {
		in.BodyB64 = base64.StdEncoding.EncodeToString(body)
	}
	return json.Marshal(in)
}

func decodeBeforeChatOut(raw []byte) ([]byte, error) {
	var out beforeChatOut
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.BodyB64 == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(out.BodyB64)
}

func encodeAfterCallIn(call *sdkhook.CallContext, info sdkhook.AfterCallInfo, config json.RawMessage) ([]byte, error) {
	in := afterCallIn{
		Principal: principalWire{
			OrganizationID: call.Principal.OrganizationID,
			ProjectID:      call.Principal.ProjectID,
			APIKeyID:       call.Principal.APIKeyID,
			Kind:           call.Principal.Kind,
			OwnerUserID:    call.Principal.OwnerUserID,
			Name:           call.Principal.Name,
		},
		Route: routeWire{
			Model:    call.Route.Model,
			Path:     call.Route.Path,
			Stream:   call.Route.Stream,
			Modality: call.Route.Modality,
		},
		Tags:      call.Tags,
		Metadata:  call.Metadata,
		Status:    info.Status,
		LatencyMs: info.LatencyMs,
		Provider:  info.ProviderType,
		Target:    info.TargetModel,
		PromptTok: info.PromptTokens,
		ComplTok:  info.CompletionTokens,
		Config:    config,
	}
	if in.Tags == nil {
		in.Tags = map[string]string{}
	}
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	return json.Marshal(in)
}
