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

func (c *RequestContext) ToHookMap() map[string]any {
	headerMap := make(map[string]string, len(c.Headers))
	for k, vals := range c.Headers {
		if len(vals) > 0 {
			headerMap[k] = vals[0]
		}
	}
	return map[string]any{
		"model":    c.Model,
		"headers":  headerMap,
		"metadata": c.Metadata,
	}
}

func (c *RequestContext) ApplyHookMap(m map[string]any) {
	if model, ok := m["model"].(string); ok && model != "" {
		c.Model = model
	}
	if headers, ok := m["headers"].(map[string]any); ok {
		for k, v := range headers {
			if s, ok := v.(string); ok {
				c.Headers.Set(k, s)
			}
		}
	}
	if meta, ok := m["metadata"].(map[string]any); ok {
		for k, v := range meta {
			c.Metadata[k] = v
		}
	}
}
