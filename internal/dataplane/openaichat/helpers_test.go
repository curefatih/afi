package openaichat

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteSSEChunkAndDone(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSSEChunk(&buf, "id1", "m", map[string]any{"content": "hi"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := WriteSSEDone(&buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "chat.completion.chunk") || !strings.Contains(s, `"content":"hi"`) {
		t.Fatalf("%s", s)
	}
	if !strings.Contains(s, "data: [DONE]") {
		t.Fatalf("missing DONE: %s", s)
	}
}
