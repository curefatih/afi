package echo

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	sdkprovider "github.com/curefatih/afi/sdk/provider"
)

func TestEchoChat(t *testing.T) {
	t.Parallel()
	a := New()
	body := []byte(`{"model":"echo-demo","messages":[{"role":"user","content":"hello world"}]}`)
	resp, err := a.Chat(context.Background(), sdkprovider.ProviderConfig{Type: Type}, "echo-demo", body, false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Choices) != 1 || out.Choices[0].Message.Content != "echo: hello world" {
		t.Fatalf("%s", raw)
	}
}

func TestEchoRejectsStream(t *testing.T) {
	t.Parallel()
	_, err := New().Chat(context.Background(), sdkprovider.ProviderConfig{}, "m", []byte(`{}`), true)
	if err == nil {
		t.Fatal("expected error")
	}
}
