package echo

import (
	"context"
	"fmt"
	"strings"

	"github.com/curefatih/afi/sdk/chatir"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
)

// Type is the snapshot provider.type string for this adapter.
const Type = "echo"

// Adapter is a no-network ChatProvider that echoes the last user message.
type Adapter struct{}

func New() sdkprovider.ChatProvider { return Adapter{} }

func (Adapter) Type() string { return Type }

func (Adapter) Capabilities() sdkprovider.Capabilities {
	return sdkprovider.Capabilities{Chat: true, Stream: false}
}

func (Adapter) ChatIR(ctx context.Context, cfg sdkprovider.ProviderConfig, targetModel string, req chatir.Request) (chatir.Result, error) {
	_ = ctx
	_ = cfg
	if req.Stream {
		return chatir.Result{}, fmt.Errorf("streaming is not supported for provider type %q", Type)
	}
	model := targetModel
	if model == "" {
		model = req.Model
	}
	userText := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userText = req.Messages[i].Content
			break
		}
	}
	content := "echo: " + userText
	return chatir.Result{
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"application/json"}},
		Response: &chatir.Response{
			ID: "chatcmpl-echo", Model: model, Role: "assistant", Content: content,
			FinishReason: "stop",
			Usage: chatir.Usage{
				PromptTokens:     int64(max(1, len(strings.Fields(userText)))),
				CompletionTokens: int64(max(1, len(strings.Fields(content)))),
			},
		},
	}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var _ sdkprovider.ChatProvider = Adapter{}
