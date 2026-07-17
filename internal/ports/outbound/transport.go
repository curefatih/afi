package outbound

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Request struct {
	Method string

	URL *url.URL

	Headers http.Header

	Body io.Reader

	Timeout time.Duration
}

type Response struct {
	StatusCode int

	Headers http.Header

	Body io.ReadCloser
}

type Transport interface {
	Do(
		ctx context.Context,
		req *Request,
	) (*Response, error)
}
