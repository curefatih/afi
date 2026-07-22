package dataplane

import "net/http"

// applyResponseHeaders merges call.ResponseHeaders onto the client response.
func applyResponseHeaders(w http.ResponseWriter, call *CallContext) {
	if call == nil || w == nil {
		return
	}
	for k, v := range call.ResponseHeaders {
		if k == "" {
			continue
		}
		w.Header().Set(http.CanonicalHeaderKey(k), v)
	}
}

// mergeRequestHeaders overlays call.RequestHeaders onto an outbound request.
func mergeRequestHeaders(dst http.Header, call *CallContext) {
	if call == nil || dst == nil {
		return
	}
	for k, v := range call.RequestHeaders {
		if k == "" {
			continue
		}
		dst.Set(http.CanonicalHeaderKey(k), v)
	}
}
