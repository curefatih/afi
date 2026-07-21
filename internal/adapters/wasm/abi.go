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
	Principal principalWire    `json:"principal"`
	Route     routeWire        `json:"route"`
	Tags      map[string]string `json:"tags"`
	Metadata  map[string]any   `json:"metadata"`
	BodyB64   string           `json:"body_b64"`
}

type beforeCallOut struct {
	Allow    bool              `json:"allow"`
	Status   int               `json:"status"`
	Reason   string            `json:"reason"`
	Message  string            `json:"message"`
	Headers  map[string]string `json:"headers"`
	Tags     map[string]string `json:"tags"`
	Metadata map[string]any    `json:"metadata"`
	BodyB64  *string           `json:"body_b64"`
}

type beforeChatIn struct {
	BodyB64 string `json:"body_b64"`
}

type beforeChatOut struct {
	BodyB64 string `json:"body_b64"`
}

func encodeBeforeCallIn(call *sdkhook.CallContext) ([]byte, error) {
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
		Tags:     call.Tags,
		Metadata: call.Metadata,
	}
	if in.Tags == nil {
		in.Tags = map[string]string{}
	}
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
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
	return d, nil
}

func encodeBeforeChatIn(body []byte) ([]byte, error) {
	in := beforeChatIn{}
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
