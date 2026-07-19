package demohook

import (
	"context"
	"encoding/json"
	"testing"
)

func TestBeforeChatPrefixesUser(t *testing.T) {
	t.Parallel()
	body := []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}]}`)
	out, err := New().BeforeChat(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	var req struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(out, &req); err != nil {
		t.Fatal(err)
	}
	if req.Messages[0].Content != "[hook:demo] hi" {
		t.Fatalf("%q", req.Messages[0].Content)
	}
}
