package routing

import (
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/types"
)

type Decision struct {
	Model            string
	Provider         string
	FallbackModel    string
	FallbackProvider string
	RuleName         string
}

type Engine struct {
	cfg *config.Config
}

type Input struct {
	Request *types.ChatCompletionRequest
	Headers http.Header
	// TODO: add rules
}

func NewEngine(cfg *config.Config) *Engine {
	return &Engine{cfg: cfg}
}

func (e *Engine) Resolve(in Input) (Decision, bool) {
	// resolves by the model name like openai/gpt-4, anthropic/claude-3.5-sonnet, etc.
	model := in.Request.Model
	parts := strings.Split(model, "/")
	if len(parts) != 2 {
		return Decision{}, false
	}
	providerName := parts[0]
	modelName := parts[1]

	if _, ok := e.cfg.Providers[providerName]; !ok {
		return Decision{}, false
	}

	return Decision{Model: modelName, Provider: providerName}, true
}
