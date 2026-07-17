package httptransport

import (
	"context"
	"net/http"

	"github.com/curefatih/afi/internal/ports/outbound"
)

type Client struct {
	client *http.Client
}

func New(client *http.Client) *Client {
	return &Client{
		client: client,
	}
}

func (c *Client) Do(
	ctx context.Context,
	req *outbound.Request,
) (*outbound.Response, error) {

	httpReq, err := http.NewRequestWithContext(
		ctx,
		req.Method,
		req.URL.String(),
		req.Body,
	)

	if err != nil {
		return nil, err
	}

	httpReq.Header = req.Headers

	resp, err := c.client.Do(httpReq)

	if err != nil {
		return nil, err
	}

	return &outbound.Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       resp.Body,
	}, nil
}
