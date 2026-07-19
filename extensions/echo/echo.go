package echo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

func (Adapter) Chat(ctx context.Context, cfg sdkprovider.ProviderConfig, targetModel string, body []byte, stream bool) (*http.Response, error) {
	_ = ctx
	_ = cfg
	if stream {
		return nil, fmt.Errorf("streaming is not supported for provider type %q", Type)
	}
	model := targetModel
	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("invalid chat body: %w", err)
	}
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
	payload := map[string]any{
		"id":      "chatcmpl-echo",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]string{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": "stop",
		}},
		"usage": map[string]int{
			"prompt_tokens":     max(1, len(strings.Fields(userText))),
			"completion_tokens": max(1, len(strings.Fields(content))),
			"total_tokens":      0,
		},
	}
	u := payload["usage"].(map[string]int)
	u["total_tokens"] = u["prompt_tokens"] + u["completion_tokens"]
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(raw)),
	}
	resp.Header.Set("Content-Type", "application/json")
	return resp, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var _ sdkprovider.ChatProvider = Adapter{}
