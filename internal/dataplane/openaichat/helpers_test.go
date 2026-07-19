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

func TestCopySSEAndParseUsage(t *testing.T) {
	src := strings.NewReader(strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"hi"}}]}`,
		``,
		`data: {"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":3}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n"))
	var dst bytes.Buffer
	prompt, completion, err := CopySSEAndParseUsage(&dst, src)
	if err != nil {
		t.Fatal(err)
	}
	if prompt != 12 || completion != 3 {
		t.Fatalf("prompt=%d completion=%d", prompt, completion)
	}
	if !strings.Contains(dst.String(), "data: [DONE]") {
		t.Fatalf("dst=%s", dst.String())
	}
}
