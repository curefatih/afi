package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrPluginExecutionTimeout = errors.New("plugin runtime execution threshold limit breached")
	ErrInvalidHookStage       = errors.New("the specified plugin registration execution stage is invalid")
)

// HookStage represents an explicit checkpoint within the Gateway request execution lifecycle.
type HookStage string

const (
	StageOnRequest            HookStage = "onRequest"
	StageOnBeforeUpstreamCall HookStage = "onBeforeUpstreamCall"
	StageOnResponse           HookStage = "onResponse"
	StageOnResponseChunk      HookStage = "onResponseChunk"
)

// IsValid checks if the string represents a supported gateway hook intercept checkpoint.
func (h HookStage) IsValid() bool {
	switch h {
	case StageOnRequest, StageOnBeforeUpstreamCall, StageOnResponse, StageOnResponseChunk:
		return true
	}
	return false
}

// RuntimeConfig dictates safety boundaries passed down to the underlying JavaScript compilation engine.
type RuntimeConfig struct {
	Timeout        time.Duration `json:"timeout"`          // Max execution budget per call (e.g., 50ms)
	MaxMemoryBytes int64         `json:"max_memory_bytes"` // Max heap allocation threshold to prevent OOM
}

// DefaultRuntimeConfig applies baseline container constraints for safe JS isolation execution.
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Timeout:        50 * time.Millisecond,
		MaxMemoryBytes: 1024 * 1024 * 5, // 5MB allocation cap
	}
}

// CustomPlugin defines the model representing a user's JavaScript extension.
type CustomPlugin struct {
	ID        string        `json:"id"`
	ProjectID string        `json:"project_id"`
	Name      string        `json:"name"`
	Stage     HookStage     `json:"stage"`
	Script    string        `json:"script"` // Raw JavaScript ECMAScript string
	IsActive  bool          `json:"is_active"`
	Config    RuntimeConfig `json:"config"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Validate ensures that a hook is structurally safe and properly aligned prior to ingestion/persistence.
func (p *CustomPlugin) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("plugin identification matrix cannot be blank")
	}
	if p.ProjectID == "" {
		return fmt.Errorf("plugin mapping missing parent structural assignment context ID")
	}
	if !p.Stage.IsValid() {
		return fmt.Errorf("%w: received %s", ErrInvalidHookStage, p.Stage)
	}
	if p.Script == "" {
		return fmt.Errorf("plugin execution runtime body script payload is blank")
	}
	if p.Config.Timeout <= 0 {
		p.Config.Timeout = DefaultRuntimeConfig().Timeout
	}
	return nil
}
