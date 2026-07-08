package routing

import (
	"testing"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/types"
)

func TestEngine_Resolve(t *testing.T) {
	engine := NewEngine(&config.Config{})
	decision, ok := engine.Resolve(Input{
		Request: &types.ChatCompletionRequest{Model: "openai/gpt-4"},
	})
	if !ok {
		t.Errorf("expected ok to be true")
	}
	if decision.Model != "gpt-4" {
		t.Errorf("expected model to be gpt-4, got %s", decision.Model)
	}
	if decision.Provider != "openai" {
		t.Errorf("expected provider to be openai, got %s", decision.Provider)
	}
}

func TestEngine_Resolve_InvalidModel(t *testing.T) {
	engine := NewEngine(&config.Config{})
	decision, ok := engine.Resolve(Input{
		Request: &types.ChatCompletionRequest{Model: "invalid/model"},
	})
	if ok {
		t.Errorf("expected ok to be false")
	}
	if decision.Model != "" {
		t.Errorf("expected model to be empty, got %s", decision.Model)
	}
	if decision.Provider != "" {
		t.Errorf("expected provider to be empty, got %s", decision.Provider)
	}
}
