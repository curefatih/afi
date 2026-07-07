package types

import (
	"encoding/json"
	"testing"
)

func TestMessageText(t *testing.T) {
	plain, _ := json.Marshal("hello")
	if got := MessageText(plain); got != "hello" {
		t.Fatalf("plain text: got %q", got)
	}

	parts, _ := json.Marshal([]map[string]string{
		{"type": "text", "text": "hello "},
		{"type": "text", "text": "world"},
	})
	if got := MessageText(parts); got != "hello world" {
		t.Fatalf("multipart text: got %q", got)
	}
}
