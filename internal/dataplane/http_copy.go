package dataplane

import (
	"fmt"
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

func (p *Pipeline) openaiAudioClient() (OpenAITransport, error) {
	if p.Providers != nil {
		if cp, ok := p.Providers.Get("openai"); ok {
			if a, ok := cp.(OpenAITransportProvider); ok {
				if t := a.OpenAITransport(); t != nil {
					return t, nil
				}
			}
		}
		if cp, ok := p.Providers.Get("openai_compatible"); ok {
			if a, ok := cp.(OpenAITransportProvider); ok {
				if t := a.OpenAITransport(); t != nil {
					return t, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("openai transport not registered")
}

func (p *Pipeline) anthropicClient() (AnthropicTransport, error) {
	if p.Providers != nil {
		if cp, ok := p.Providers.Get("anthropic"); ok {
			if a, ok := cp.(AnthropicTransportProvider); ok {
				if t := a.AnthropicTransport(); t != nil {
					return t, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("anthropic transport not registered")
}
