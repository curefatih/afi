package dataplane

import (
	"io"
	"net/http"
	"strings"
)

// CopyResponse copies an upstream response to the client writer.
func CopyResponse(w http.ResponseWriter, resp *http.Response) error {
	for k, vals := range resp.Header {
		if strings.EqualFold(k, "Transfer-Encoding") || strings.EqualFold(k, "Connection") {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err := io.Copy(w, resp.Body)
	return err
}
