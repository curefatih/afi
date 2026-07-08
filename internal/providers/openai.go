package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/types"
)

const defaultHTTPTimeout = 120 * time.Second

// OpenAI is an OpenAI-compatible passthrough adapter.
type OpenAI struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewOpenAI(name string, cfg config.Provider) *OpenAI {
	return &OpenAI{
		name:    name,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		client:  &http.Client{Timeout: defaultHTTPTimeout},
	}
}

func (o *OpenAI) Name() string { return o.name }

func (o *OpenAI) UpstreamChatCompletion(ctx context.Context, req *types.ChatCompletionRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	upstream, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	upstream.Header.Set("Content-Type", "application/json")
	upstream.Header.Set("Authorization", "Bearer "+o.apiKey)
	if req.Stream {
		upstream.Header.Set("Accept", "text/event-stream")
	}

	return o.client.Do(upstream)
}

func (o *OpenAI) WriteResponse(resp *http.Response, _ *types.ChatCompletionRequest, w http.ResponseWriter) error {
	defer resp.Body.Close()
	return copyUpstreamResponse(w, resp)
}

func (o *OpenAI) Passthrough(ctx context.Context, method, path string, body []byte, headers http.Header, w http.ResponseWriter) error {
	req, err := http.NewRequestWithContext(ctx, method, o.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build passthrough request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	if ct := headers.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("upstream request: %w", err)
	}
	return copyUpstreamResponse(w, resp)
}

func copyUpstreamResponse(w http.ResponseWriter, resp *http.Response) error {
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		slog.Debug("upstream error", "status", resp.StatusCode, "body", string(body))
		_, err := w.Write(body)
		return err
	}

	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming not supported")
		}
		buf := make([]byte, 32*1024)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					return writeErr
				}
				flusher.Flush()
			}
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}

	_, err := io.Copy(w, resp.Body)
	return err
}
