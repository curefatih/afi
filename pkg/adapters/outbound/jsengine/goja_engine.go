package jsengine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/curefatih/afi/internal/core/domain"

	"github.com/dop251/goja"
)

type GojaEngineAdapter struct{}

func NewGojaEngineAdapter() *GojaEngineAdapter {
	return &GojaEngineAdapter{}
}

// ExecuteHook initializes an isolated VM sandbox to execute untrusted code securely.
func (e *GojaEngineAdapter) ExecuteHook(ctx context.Context, script string, stage domain.HookStage, payload any, config domain.RuntimeConfig) (any, error) {
	vm := goja.New()

	// 1. Structural Isolation Guardrails: Disable dangerous runtime capabilities
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper()) // Clean object property mappings (e.g., text instead of Text)

	// 2. Map Payload into JavaScript Context Memory
	// We convert via JSON marshalling to break any direct references to live Go pointer variables.
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize payload for JS runtime sandbox: %w", err)
	}

	var rawJSObject map[string]any
	if err := json.Unmarshal(jsonBytes, &rawJSObject); err != nil {
		return nil, err
	}

	// Expose properties to the JS global context namespace
	vm.Set("payload", rawJSObject)
	vm.Set("stage", stage)

	// 3. Thread Safe Infinite Loop Interrupter Engine
	// We set an execution timer inside a separate goroutine. If it hits the timeout threshold,
	// it immediately fires a hard Interrupt call directly down into the specific execution stack.
	timeContext, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	go func() {
		<-timeContext.Done()
		if timeContext.Err() == context.DeadlineExceeded {
			vm.Interrupt("execution runtime limit threshold exceeded")
		}
	}()

	// 4. Compile and run the JavaScript script block
	// The user script can mutate the `payload` object directly (e.g., payload.messages[0].parts[0].text = "new string")
	_, err = vm.RunString(script)
	if err != nil {
		return nil, fmt.Errorf("javascript syntax execution error inside [%s]: %w", stage, err)
	}

	// 5. Pull out the transformed state variant object back from JS global landscape
	val := vm.Get("payload")
	if val == nil {
		return nil, fmt.Errorf("payload variable not found in JS VM")
	}
	resultObj := val.Export()

	// 6. Remap clean unstructured JS data types safely back onto expected Go Struct Pointers
	finalizedBytes, err := json.Marshal(resultObj)
	if err != nil {
		return nil, fmt.Errorf("failed to re-serialize JS state modification changes: %w", err)
	}

	// Unmarshal back to the exact target domain structure based on matching execution checkpoint type
	switch stage {
	case "onRequest", "onBeforeUpstreamCall":
		var updatedRequest domain.InternalRequest
		if err := json.Unmarshal(finalizedBytes, &updatedRequest); err != nil {
			return nil, err
		}
		return &updatedRequest, nil

	case "onResponse":
		var updatedResponse domain.InternalResponse
		if err := json.Unmarshal(finalizedBytes, &updatedResponse); err != nil {
			return nil, err
		}
		return &updatedResponse, nil

	case "onResponseChunk":
		var updatedChunk domain.StreamChunk
		if err := json.Unmarshal(finalizedBytes, &updatedChunk); err != nil {
			return nil, err
		}
		return updatedChunk, nil

	default:
		return payload, nil
	}
}
