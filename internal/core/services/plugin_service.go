// internal/core/services/plugin_service.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/core/domain"
)

type PluginRepository interface {
	SavePlugin(ctx context.Context, plugin *domain.CustomPlugin) error
	GetActivePlugin(ctx context.Context, projectID string, stage domain.HookStage) (*domain.CustomPlugin, error)
}

type PluginService struct {
	repo PluginRepository
}

func NewPluginService(repo PluginRepository) *PluginService {
	return &PluginService{repo: repo}
}

func (s *PluginService) SaveHook(ctx context.Context, projectID string, stage domain.HookStage, script string) error {
	plugin := &domain.CustomPlugin{
		ID:        fmt.Sprintf("plg_%d", time.Now().UnixNano()),
		ProjectID: projectID,
		Name:      fmt.Sprintf("%s_hook", stage),
		Stage:     stage,
		Script:    script,
		IsActive:  true,
		Config:    domain.DefaultRuntimeConfig(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := plugin.Validate(); err != nil {
		return fmt.Errorf("plugin validation constraint matrix failed: %w", err)
	}

	return s.repo.SavePlugin(ctx, plugin)
}

func (s *PluginService) GetHook(ctx context.Context, projectID string, stage string) (*domain.CustomPlugin, bool) {
	plugin, err := s.repo.GetActivePlugin(ctx, projectID, domain.HookStage(stage))
	if err != nil || plugin == nil || !plugin.IsActive {
		return nil, false
	}
	return plugin, true
}
