// Package dialect encodes/decodes client wire formats against chat IR.
package dialect

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/curefatih/afi/internal/dataplane/ir"
)

// Codec converts between a client dialect and chat IR.
type Codec interface {
	Name() ir.Dialect
	DecodeRequest(body []byte) (ir.ChatRequest, error)
	EncodeResponse(resp ir.ChatResponse) ([]byte, error)
	WriteStream(w io.Writer, events <-chan ir.StreamEvent) (prompt, completion int64, err error)
}

// For returns the codec for a dialect.
func For(d ir.Dialect) (Codec, error) {
	switch d {
	case ir.DialectOpenAI:
		return OpenAI{}, nil
	case ir.DialectAnthropic:
		return Anthropic{}, nil
	default:
		return nil, fmt.Errorf("unsupported dialect %q", d)
	}
}

// WriteError writes a dialect-shaped JSON error.
func WriteError(w http.ResponseWriter, d ir.Dialect, status int, message, typ string) {
	if typ == "" {
		typ = "invalid_request_error"
	}
	typ = normalizeErrorType(d, typ)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	var payload any
	switch d {
	case ir.DialectAnthropic:
		payload = map[string]any{
			"type": "error",
			"error": map[string]string{
				"type":    typ,
				"message": message,
			},
		}
	default:
		payload = map[string]any{
			"error": map[string]string{
				"message": message,
				"type":    typ,
				"code":    typ,
			},
		}
	}
	_ = json.NewEncoder(w).Encode(payload)
}
