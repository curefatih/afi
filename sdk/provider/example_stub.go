package provider

import (
	"context"
	"fmt"
	"net/http"
)

// ExampleAdapter is a documentation stub showing the shape of a custom adapter.
// It is not registered by the gateway.
type ExampleAdapter struct{}

func (ExampleAdapter) Type() string { return "example" }

func (ExampleAdapter) Capabilities() Capabilities {
	return Capabilities{Chat: true, Stream: false}
}

func (ExampleAdapter) Chat(ctx context.Context, cfg ProviderConfig, targetModel string, body []byte, stream bool) (*http.Response, error) {
	_ = ctx
	_ = cfg
	_ = targetModel
	_ = body
	if stream {
		return nil, fmt.Errorf("streaming is not supported for provider type %q", "example")
	}
	return nil, fmt.Errorf("example adapter is a documentation stub only")
}

// Ensure ExampleAdapter satisfies ChatProvider at compile time.
var _ ChatProvider = ExampleAdapter{}
