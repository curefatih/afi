package hook

import "net/http"

// SetRequestHeader sets an outbound upstream header on the call (after BeforeCall).
func (c *CallContext) SetRequestHeader(key, value string) {
	if c == nil || key == "" {
		return
	}
	if c.RequestHeaders == nil {
		c.RequestHeaders = map[string]string{}
	}
	c.RequestHeaders[http.CanonicalHeaderKey(key)] = value
}

// SetResponseHeader sets a header to merge onto the client HTTP response.
func (c *CallContext) SetResponseHeader(key, value string) {
	if c == nil || key == "" {
		return
	}
	if c.ResponseHeaders == nil {
		c.ResponseHeaders = map[string]string{}
	}
	c.ResponseHeaders[http.CanonicalHeaderKey(key)] = value
}

// DeleteRequestHeader removes an outbound upstream header overlay.
func (c *CallContext) DeleteRequestHeader(key string) {
	if c == nil || c.RequestHeaders == nil || key == "" {
		return
	}
	delete(c.RequestHeaders, http.CanonicalHeaderKey(key))
	delete(c.RequestHeaders, key)
}

// DeleteResponseHeader removes a client response header overlay.
func (c *CallContext) DeleteResponseHeader(key string) {
	if c == nil || c.ResponseHeaders == nil || key == "" {
		return
	}
	delete(c.ResponseHeaders, http.CanonicalHeaderKey(key))
	delete(c.ResponseHeaders, key)
}
