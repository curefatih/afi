package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/curefatih/afi/internal/core/provider"
	"github.com/curefatih/afi/internal/ports/outbound"
)

type Client struct {
	cfg Config

	transport outbound.Transport

	translator *Translator
}

func NewClient(
	cfg Config,
	transport outbound.Transport,
) *Client {

	return &Client{
		cfg:        cfg,
		transport:  transport,
		translator: NewTranslator(),
	}
}

func (c *Client) Execute(
	ctx context.Context,
	req *provider.Request,
) (*provider.Response, error) {

	dto, err := c.translator.Encode(req)

	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(dto)

	if err != nil {
		return nil, err
	}

	url := c.cfg.BaseURL.ResolveReference(
		&url.URL{
			Path: "/v1/chat/completions",
		},
	)

	resp, err := c.transport.Do(ctx, &outbound.Request{
		Method: http.MethodPost,
		URL:    url,
		Headers: http.Header{
			"Authorization": []string{"Bearer " + c.cfg.APIKey},
			"Content-Type":  []string{"application/json"},
		},
		Body: bytes.NewReader(body),
	})

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, ErrUnexpectedStatus
	}

	openaiResp, err := DecodeResponse(resp.Body)

	if err != nil {
		return nil, err
	}

	return c.translator.Decode(openaiResp)
}
