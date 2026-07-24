package demohook

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/curefatih/afi/sdk/chatir"
	sdkhook "github.com/curefatih/afi/sdk/hook"
)

func TestBeforeChatPrefixesLastUserMessage(t *testing.T) {
	t.Parallel()
	req := chatir.Request{
		Model: "m",
		Messages: []chatir.Message{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "reply"},
			{Role: "user", Content: "hi"},
		},
	}
	out, err := New().BeforeChat(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if out.Messages[2].Content != "[hook:demo] hi" {
		t.Fatalf("last user=%q", out.Messages[2].Content)
	}
	if out.Messages[0].Content != "first" || out.Messages[1].Content != "reply" {
		t.Fatalf("earlier messages mutated: %+v", out.Messages)
	}
}

func TestBeforeChatIsIdempotent(t *testing.T) {
	t.Parallel()
	req := chatir.Request{Messages: []chatir.Message{{Role: "user", Content: "hi"}}}
	first, err := New().BeforeChat(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := New().BeforeChat(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	if second.Messages[0].Content != "[hook:demo] hi" {
		t.Fatalf("content=%q", second.Messages[0].Content)
	}
}

func TestBeforeChatWithoutUserMessage(t *testing.T) {
	t.Parallel()
	req := chatir.Request{Messages: []chatir.Message{{Role: "system", Content: "sys"}}}
	out, err := New().BeforeChat(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if out.Messages[0].Content != "sys" {
		t.Fatalf("content=%q", out.Messages[0].Content)
	}
}

func TestAfterChatLogsAttempt(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := NewWithLog(slog.New(slog.NewTextHandler(&buf, nil)))
	err := h.AfterChat(context.Background(), sdkhook.AfterChatInfo{
		Model:        "gpt-4o",
		Status:       "ok",
		LatencyMs:    12,
		ProviderType: "echo",
		TargetModel:  "echo-demo",
		Dialect:      "openai",
		Modality:     "chat",
	})
	if err != nil {
		t.Fatal(err)
	}
	logged := buf.String()
	for _, want := range []string{"demohook.after_chat", "gpt-4o", "echo-demo", "openai", "chat"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log missing %q: %s", want, logged)
		}
	}
}

func TestHookName(t *testing.T) {
	t.Parallel()
	if New().Name() != Name {
		t.Fatalf("name=%q", New().Name())
	}
}
