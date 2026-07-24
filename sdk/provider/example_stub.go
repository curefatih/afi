package provider

import (
	"context"
	"fmt"

	"github.com/curefatih/afi/sdk/chatir"
)

// ExampleAdapter is a documentation stub showing the shape of a custom adapter.
// Prefer the working extensions/echo package for a real registration example.
type ExampleAdapter struct{}

func (ExampleAdapter) Type() string { return "example" }

func (ExampleAdapter) Capabilities() Capabilities {
	return Capabilities{Chat: true, Stream: false}
}

func (ExampleAdapter) ChatIR(ctx context.Context, cfg ProviderConfig, targetModel string, req chatir.Request) (chatir.Result, error) {
	_ = ctx
	_ = cfg
	_ = targetModel
	if req.Stream {
		return chatir.Result{}, fmt.Errorf("streaming is not supported for provider type %q", "example")
	}
	return chatir.Result{}, fmt.Errorf("example adapter is a documentation stub only; use extensions/echo")
}

// Ensure ExampleAdapter satisfies ChatProvider at compile time.
var _ ChatProvider = ExampleAdapter{}
