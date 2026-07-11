package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
)

var (
	ErrProviderNotFound = errors.New("requested outbound provider client not found")
)

type GatewayService struct {
	jsEngine      ports.JSEngine
	pluginService ports.PluginService        // Service to fetch scripts from DB/Cache
	budgetService ports.BudgetService        // Orchestrates multi-checkpoint evaluations
	routerService ports.RouterService        // Matches rules to get domain.TargetDestination
	providers     map[string]ports.LLMClient // Keyed by provider string like "openai", "anthropic"
}

func NewGatewayService(
	jsEngine ports.JSEngine,
	pluginService ports.PluginService,
	budgetService ports.BudgetService,
	routerService ports.RouterService,
	providers map[string]ports.LLMClient,
) *GatewayService {
	return &GatewayService{
		jsEngine:      jsEngine,
		pluginService: pluginService,
		budgetService: budgetService,
		routerService: routerService,
		providers:     providers,
	}
}

// ExecuteUnary processes a non-streaming, standard request/response lifecycle.
func (s *GatewayService) ExecuteUnary(ctx context.Context, req *domain.InternalRequest) (*domain.InternalResponse, error) {
	// 1. Snapshot the untamperable system context parameters securely
	systemMetadataBackup := req.Metadata

	if plugin, ok := s.pluginService.GetHook(ctx, req.Metadata.ProjectID, domain.StageOnRequest); ok {
		mutated, err := s.jsEngine.ExecuteHook(ctx, plugin.Script, domain.StageOnRequest, req, plugin.Config)
		if err != nil {
			return nil, fmt.Errorf("onRequest hook execution failure: %w", err)
		}
		if incoming, validation := mutated.(*domain.InternalRequest); validation {
			req = incoming
			// 2. Force-restore the context metadata so JavaScript code can never drop or forge it
			req.Metadata = systemMetadataBackup
		}
	}

	// Budget & Router Verification Checklist Matrix
	if err := s.budgetService.Check(ctx, req.Metadata); err != nil {
		return nil, fmt.Errorf("budget enforcement block: %w", err)
	}

	target, err := s.routerService.Route(req)
	if err != nil {
		return nil, fmt.Errorf("routing resolution matrix failed: %w", err)
	}

	req.Model = target.TargetModel
	providerClient, exists := s.providers[target.Provider]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, target.Provider)
	}

	if plugin, ok := s.pluginService.GetHook(ctx, req.Metadata.ProjectID, domain.StageOnBeforeUpstreamCall); ok {
		mutated, err := s.jsEngine.ExecuteHook(ctx, plugin.Script, domain.StageOnBeforeUpstreamCall, req, plugin.Config)
		if err != nil {
			return nil, fmt.Errorf("onBeforeUpstreamCall hook execution failure: %w", err)
		}
		if incoming, validation := mutated.(*domain.InternalRequest); validation {
			req = incoming
		}
	}

	// Execute Actual Upstream Network Operation
	resp, err := providerClient.Call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("upstream destination execution error: %w", err)
	}

	if plugin, ok := s.pluginService.GetHook(ctx, req.Metadata.ProjectID, domain.StageOnResponse); ok {
		mutated, err := s.jsEngine.ExecuteHook(ctx, plugin.Script, domain.StageOnResponse, resp, plugin.Config)
		if err != nil {
			return nil, fmt.Errorf("onResponse hook execution failure: %w", err)
		}
		if outgoing, validation := mutated.(*domain.InternalResponse); validation {
			resp = outgoing
		}
	}

	go s.budgetService.CommitUsage(context.Background(), req.Metadata, resp.Usage)

	return resp, nil
}

// ExecuteStream pipes real-time streaming chunks back safely without thread blocking.

func (s *GatewayService) ExecuteStream(ctx context.Context, req *domain.InternalRequest) (<-chan domain.StreamChunk, <-chan error) {
	outChunks := make(chan domain.StreamChunk)
	outErr := make(chan error, 1)

	// 1. Securely snapshot the authenticated context properties
	systemMetadataBackup := req.Metadata

	if plugin, ok := s.pluginService.GetHook(ctx, req.Metadata.ProjectID, domain.StageOnRequest); ok {
		mutated, err := s.jsEngine.ExecuteHook(ctx, plugin.Script, domain.StageOnRequest, req, plugin.Config)
		if err != nil {
			outErr <- fmt.Errorf("onRequest streaming hook failure: %w", err)
			close(outChunks)
			close(outErr)
			return outChunks, outErr
		}
		if incoming, validation := mutated.(*domain.InternalRequest); validation {
			req = incoming
			// 2. Force-restore context parameters back onto the mutated object
			req.Metadata = systemMetadataBackup
		}
	}

	// Budget Check
	if err := s.budgetService.Check(ctx, req.Metadata); err != nil {
		outErr <- fmt.Errorf("budget blocking error: %w", err)
		close(outChunks)
		close(outErr)
		return outChunks, outErr
	}

	// Route Resolution
	target, err := s.routerService.Route(req)
	if err != nil {
		outErr <- fmt.Errorf("routing streaming target failed: %w", err)
		close(outChunks)
		close(outErr)
		return outChunks, outErr
	}

	req.Model = target.TargetModel
	providerClient, exists := s.providers[target.Provider]
	if !exists {
		outErr <- fmt.Errorf("provider match missing: %s", target.Provider)
		close(outChunks)
		close(outErr)
		return outChunks, outErr
	}

	if plugin, ok := s.pluginService.GetHook(ctx, req.Metadata.ProjectID, domain.StageOnBeforeUpstreamCall); ok {
		mutated, err := s.jsEngine.ExecuteHook(ctx, plugin.Script, domain.StageOnBeforeUpstreamCall, req, plugin.Config)
		if err != nil {
			outErr <- fmt.Errorf("onBeforeUpstreamCall streaming hook failure: %w", err)
			close(outChunks)
			close(outErr)
			return outChunks, outErr
		}
		if incoming, validation := mutated.(*domain.InternalRequest); validation {
			req = incoming
			// 3. Keep metadata completely intact right before upstream dispatch
			req.Metadata = systemMetadataBackup
		}
	}

	// 4. Initialize Core Upstream SSE Stream Pipeline Channel Link
	vendorChunks, vendorErrCh := providerClient.StreamCall(ctx, req)

	go func() {
		defer close(outChunks)
		defer close(outErr)

		var finalUsage domain.TokenUsage

		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-vendorErrCh:
				if ok && err != nil {
					outErr <- err
				}
				return
			case chunk, ok := <-vendorChunks:
				if !ok {
					fmt.Printf("[GATEWAY TRACE] Channel closed. Final total tracking tokens gathered: %d\n", finalUsage.TotalTokens)
					if finalUsage.TotalTokens > 0 {
						s.budgetService.CommitUsage(context.Background(), systemMetadataBackup, finalUsage)
					}
					return
				}

				if chunk.Usage.TotalTokens > 0 {
					finalUsage = chunk.Usage
					fmt.Printf("[GATEWAY TRACE] Captured usage snapshot directly from chunk: %d tokens\n", finalUsage.TotalTokens)
				}

				if plugin, ok := s.pluginService.GetHook(ctx, systemMetadataBackup.ProjectID, domain.StageOnResponseChunk); ok {
					mutated, err := s.jsEngine.ExecuteHook(ctx, plugin.Script, domain.StageOnResponseChunk, chunk, plugin.Config)
					if err == nil {
						if processedChunk, validation := mutated.(domain.StreamChunk); validation {
							chunk = processedChunk
						}
					}
				}

				outChunks <- chunk
			}
		}
	}()

	return outChunks, outErr

}

// Helper calculation technique if providers do not pass analytical metadata via stream endings
func (s *GatewayService) estimateStreamingTokens(model string, content string) domain.TokenUsage {
	// Abstracted representation: evaluate character heuristics or call sub-token infrastructure weights.
	words := len(strings.Fields(content))
	estimatedTokens := int(float64(words) * 1.3)

	return domain.TokenUsage{
		OutputTokens: estimatedTokens,
		TotalTokens:  estimatedTokens,
	}
}
