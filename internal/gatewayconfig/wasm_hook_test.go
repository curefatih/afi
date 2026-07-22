package gatewayconfig

import (
	"encoding/json"
	"testing"
)

func TestNewWasmHook(t *testing.T) {
	h, err := NewWasmHook("w1", "o1", "block", WasmPhaseBeforeCall, "file:///tmp/h.wasm", "abc", true, 10, json.RawMessage(`{"x":1}`), timeNowUTC())
	if err != nil {
		t.Fatal(err)
	}
	if h.Phase != WasmPhaseBeforeCall || h.Name != "block" {
		t.Fatalf("%+v", h)
	}
}

func TestNewWasmHookBadPhase(t *testing.T) {
	_, err := NewWasmHook("w1", "o1", "n", "nope", "/x.wasm", "", true, 1, nil, timeNowUTC())
	if err == nil {
		t.Fatal("expected error")
	}
}
