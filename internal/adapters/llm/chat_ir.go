package llm

import (
	"context"
	"io"
	"net/http"

	"github.com/curefatih/afi/internal/dataplane/dialect"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
)

// ChatIR runs chat against an OpenAI-compatible upstream using IR.
func (c *OpenAIClient) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	body, err := ir.EncodeOpenAI(req)
	if err != nil {
		return ir.ChatResult{}, err
	}
	resp, err := c.ChatCompletions(ctx, provider, targetModel, body, req.Stream)
	if err != nil {
		return ir.ChatResult{}, err
	}
	return mapHTTPToChatResult(resp, req.Stream, ir.DecodeOpenAIResponse, dialect.ParseOpenAISSE)
}

// ChatIR runs chat against Anthropic Messages using IR.
func (c *AnthropicClient) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	body, err := ir.EncodeAnthropic(req, targetModel)
	if err != nil {
		return ir.ChatResult{}, err
	}
	resp, err := c.PassThrough(ctx, provider, targetModel, body, req.Stream)
	if err != nil {
		return ir.ChatResult{}, err
	}
	return mapHTTPToChatResult(resp, req.Stream, ir.DecodeAnthropicResponse, dialect.ParseAnthropicSSE)
}

// ChatIR runs chat against Gemini generateContent using IR.
func (c *GeminiClient) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	body, err := irToGemini(req)
	if err != nil {
		return ir.ChatResult{}, err
	}
	// GenerateContent expects OpenAI body; use a dedicated IR path instead.
	resp, err := c.generateContentIR(ctx, provider, targetModel, body, req.Stream)
	if err != nil {
		return ir.ChatResult{}, err
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, ErrorBody: raw}, nil
	}
	if req.Stream {
		ch := parseGeminiSSEToIR(resp.Body, targetModel)
		return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, Events: wrapClose(ch, resp.Body)}, nil
	}
	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return ir.ChatResult{}, err
	}
	mapped, err := geminiJSONToIR(raw, targetModel)
	if err != nil {
		return ir.ChatResult{}, err
	}
	return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, Response: &mapped}, nil
}

type streamParser func(io.Reader) <-chan ir.StreamEvent

func mapHTTPToChatResult(resp *http.Response, stream bool, parse func([]byte) (ir.ChatResponse, error), parseStream streamParser) (ir.ChatResult, error) {
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, ErrorBody: raw}, nil
	}
	if stream {
		ch := parseStream(resp.Body)
		return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, Events: wrapClose(ch, resp.Body)}, nil
	}
	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return ir.ChatResult{}, err
	}
	mapped, err := parse(raw)
	if err != nil {
		return ir.ChatResult{}, err
	}
	return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, Response: &mapped}, nil
}

func wrapClose(ch <-chan ir.StreamEvent, body io.Closer) <-chan ir.StreamEvent {
	out := make(chan ir.StreamEvent, 16)
	go func() {
		defer close(out)
		defer body.Close()
		for ev := range ch {
			out <- ev
		}
	}()
	return out
}
