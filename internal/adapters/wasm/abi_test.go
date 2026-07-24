package wasm

import (
	"encoding/json"
	"testing"

	"github.com/curefatih/afi/sdk/chatir"
	sdkhook "github.com/curefatih/afi/sdk/hook"
)

func TestEncodeDecodeBeforeCall(t *testing.T) {
	call := &sdkhook.CallContext{
		Principal: sdkhook.Principal{OrganizationID: "org1", ProjectID: "p1", APIKeyID: "k1"},
		Route:     sdkhook.RouteContext{Model: "gpt", Path: "/v1/chat/completions", Modality: "chat"},
		Tags:      map[string]string{"plan": "enterprise"},
		Metadata:  map[string]any{"x": float64(1)},
		Body:      []byte(`{"messages":[]}`),
	}
	raw, err := encodeBeforeCallIn(call, nil)
	if err != nil {
		t.Fatal(err)
	}
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatal(err)
	}
	if probe["body_b64"] == "" {
		t.Fatal("expected body_b64")
	}

	outJSON := []byte(`{"allow":true,"tags":{"plan":"enterprise","wasm":"1"},"metadata":{"x":1,"y":2},"body_b64":null}`)
	d, err := applyBeforeCallOut(call, outJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Allow {
		t.Fatal("expected allow")
	}
	if call.Tags["wasm"] != "1" {
		t.Fatalf("tags=%v", call.Tags)
	}
	if call.Metadata["y"].(float64) != 2 {
		t.Fatalf("metadata=%v", call.Metadata)
	}
}

func TestApplyBeforeCallDeny(t *testing.T) {
	call := &sdkhook.CallContext{Tags: map[string]string{}, Metadata: map[string]any{}}
	d, err := applyBeforeCallOut(call, []byte(`{"allow":false,"reason":"blocked","message":"no"}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Allow || d.Status != 403 || d.Reason != "blocked" {
		t.Fatalf("got %+v", d)
	}
}

func TestBeforeChatRoundTrip(t *testing.T) {
	req := chatir.Request{
		Model: "route",
		Messages: []chatir.Message{{
			Role: "user", Content: "hi",
			Parts: []chatir.ContentPart{{
				Type:  chatir.ContentImage,
				Image: &chatir.ImageSource{MediaType: "image/png", Data: "aW1hZ2U="},
			}},
		}},
		Tools: []chatir.Tool{{
			Name: "lookup", Parameters: json.RawMessage(`{"type":"object"}`),
		}},
	}
	raw, err := encodeBeforeChatIn(req, json.RawMessage(`{"prefix":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["request"].(map[string]any)["model"] != "route" {
		t.Fatalf("wire=%v", wire)
	}
	if wire["config"].(map[string]any)["prefix"] != "x" {
		t.Fatalf("config not passed to guest: %v", wire)
	}
	out, err := decodeBeforeChatOut(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Model != "route" || len(out.Messages) != 1 ||
		len(out.Messages[0].Parts) != 1 || len(out.Tools) != 1 {
		t.Fatalf("request=%+v", out)
	}
	if out.Messages[0].Parts[0].Image.MediaType != "image/png" {
		t.Fatalf("image part lost: %+v", out.Messages[0].Parts[0])
	}
}

func TestBeforeChatOmitsEmptyConfig(t *testing.T) {
	raw, err := encodeBeforeChatIn(chatir.Request{Model: "m"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["config"]; ok {
		t.Fatalf("expected config omitted: %s", raw)
	}
}

func TestDecodeBeforeChatOutRejectsInvalidJSON(t *testing.T) {
	if _, err := decodeBeforeChatOut([]byte(`{"request":`)); err == nil {
		t.Fatal("expected error")
	}
}
