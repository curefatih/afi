package wasm

import (
	"encoding/json"
	"testing"

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
	raw, err := encodeBeforeCallIn(call)
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
	in, err := encodeBeforeChatIn([]byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := decodeBeforeChatOut(in) // same shape body_b64
	if err != nil {
		t.Fatal(err)
	}
	// encodeBeforeChatIn produces {"body_b64":"..."} which decodeBeforeChatOut accepts
	if string(out) != `{"a":1}` {
		t.Fatalf("got %q", out)
	}
}
