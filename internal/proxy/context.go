package proxy

import (
	"encoding/json"
	"net/http"
)

// RequestContext carries request state through the hook pipeline.
type RequestContext struct {
	Model    string
	Headers  http.Header
	Body     json.RawMessage
	Metadata map[string]any
}

func NewRequestContext(model string, headers http.Header, body []byte) *RequestContext {
	h := headers.Clone()
	if h == nil {
		h = make(http.Header)
	}
	return &RequestContext{
		Model:    model,
		Headers:  h,
		Body:     json.RawMessage(body),
		Metadata: make(map[string]any),
	}
}
