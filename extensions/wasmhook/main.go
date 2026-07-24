//go:build tinygo

// TinyGo WASM lifecycle hook example for AFI.
//
// Build (from this directory):
//
//	tinygo build -o hook.wasm -scheduler=none -target=wasip1 -buildmode=c-shared .
//
// Behavior:
//   - before_call: deny when tags["plan"] == "blocked"; otherwise allow and set metadata["wasm_hook"]="1"
//   - before_chat: prefix last user message content with "[wasm] " (typed chat IR JSON)
//   - after_call: no-op
package main

// #include <stdlib.h>
import "C"

import (
	"encoding/json"
	"unsafe"

	"github.com/curefatih/afi/sdk/chatir"
)

func main() {}

type principal struct {
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	APIKeyID       string `json:"api_key_id"`
	Kind           string `json:"kind"`
	OwnerUserID    string `json:"owner_user_id"`
	Name           string `json:"name"`
}

type route struct {
	Model    string `json:"model"`
	Path     string `json:"path"`
	Stream   bool   `json:"stream"`
	Modality string `json:"modality"`
}

type beforeCallIn struct {
	Principal       principal         `json:"principal"`
	Route           route             `json:"route"`
	Tags            map[string]string `json:"tags"`
	Metadata        map[string]any    `json:"metadata"`
	BodyB64         string            `json:"body_b64"`
	RequestHeaders  map[string]string `json:"request_headers"`
	ResponseHeaders map[string]string `json:"response_headers"`
}

type beforeCallOut struct {
	Allow           bool              `json:"allow"`
	Status          int               `json:"status,omitempty"`
	Reason          string            `json:"reason,omitempty"`
	Message         string            `json:"message,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	Tags            map[string]string `json:"tags,omitempty"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
	BodyB64         *string           `json:"body_b64,omitempty"`
	RequestHeaders  map[string]string `json:"request_headers,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
}

type beforeChatWire struct {
	Request chatir.Request `json:"request"`
}

//go:wasmexport before_call
func _before_call(ptr, size uint32) uint64 {
	in := ptrToBytes(ptr, size)
	var req beforeCallIn
	if err := json.Unmarshal(in, &req); err != nil {
		return leakJSON(beforeCallOut{
			Allow:   false,
			Status:  500,
			Reason:  "wasm_error",
			Message: "invalid before_call input",
		})
	}
	if req.Tags == nil {
		req.Tags = map[string]string{}
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	if req.Tags["plan"] == "blocked" {
		return leakJSON(beforeCallOut{
			Allow:    false,
			Status:   403,
			Reason:   "plan_blocked",
			Message:  "plan=blocked denied by wasm hook",
			Tags:     req.Tags,
			Metadata: req.Metadata,
		})
	}
	req.Metadata["wasm_hook"] = "1"
	if req.ResponseHeaders == nil {
		req.ResponseHeaders = map[string]string{}
	}
	req.ResponseHeaders["X-AFI-Wasm"] = "1"
	return leakJSON(beforeCallOut{
		Allow:           true,
		Tags:            req.Tags,
		Metadata:        req.Metadata,
		ResponseHeaders: req.ResponseHeaders,
	})
}

//go:wasmexport before_chat
func _before_chat(ptr, size uint32) uint64 {
	var in beforeChatWire
	if err := json.Unmarshal(ptrToBytes(ptr, size), &in); err != nil {
		return 0
	}
	for i := len(in.Request.Messages) - 1; i >= 0; i-- {
		if in.Request.Messages[i].Role != "user" {
			continue
		}
		const prefix = "[wasm] "
		content := in.Request.Messages[i].Content
		if len(content) < len(prefix) || content[:len(prefix)] != prefix {
			in.Request.Messages[i].Content = prefix + content
		}
		break
	}
	return leakJSON(in)
}

//go:wasmexport after_call
func _after_call(ptr, size uint32) uint64 {
	_ = ptrToBytes(ptr, size)
	return 0
}

func leakJSON(v any) uint64 {
	b, err := json.Marshal(v)
	if err != nil {
		b = []byte(`{"allow":false,"status":500,"reason":"wasm_error"}`)
	}
	ptr, size := stringToLeakedPtr(string(b))
	return (uint64(ptr) << 32) | uint64(size)
}

func ptrToBytes(ptr, size uint32) []byte {
	if size == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
}

func stringToLeakedPtr(s string) (uint32, uint32) {
	size := C.ulong(len(s))
	if size == 0 {
		return 0, 0
	}
	ptr := unsafe.Pointer(C.malloc(size))
	copy(unsafe.Slice((*byte)(ptr), size), s)
	return uint32(uintptr(ptr)), uint32(size)
}
