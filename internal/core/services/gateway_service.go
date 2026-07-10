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
	pluginService ports.PluginService        // Mockable service to fetch scripts from DB/Cache
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
	// ----------------------------------------------------
	// HOOK 1: onRequest
	// ----------------------------------------------------
	if script, ok := s.pluginService.GetHook(req.Metadata.ProjectID, "onRequest"); ok {
		mutated, err := s.jsEngine.ExecuteHook(ctx, script, "onRequest", req)
		if err != nil {
			return nil, fmt.Errorf("onRequest hook execution failure: %w", err)
		}
		if incoming, validation := mutated.(*domain.InternalRequest); validation {
			req = incoming
		}
	}

	// ----------------------------------------------------
	// PRE-FLIGHT CHECK: Hierarchical Budgets & Routing
	// ----------------------------------------------------
	if err := s.budgetService.Check(ctx, req.Metadata); err != nil {
		return nil, fmt.Errorf("budget enforcement block: %w", err)
	}

	target, err := s.routerService.Route(req)
	if err != nil {
		return nil, fmt.Errorf("routing resolution matrix failed: %w", err)
	}

	// Mutate model signature to map downstream target requirements
	req.Model = target.TargetModel
	providerClient, exists := s.providers[target.Provider]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, target.Provider)
	}

	// ----------------------------------------------------
	// HOOK 2: onBeforeUpstreamCall
	// ----------------------------------------------------
	if script, ok := s.pluginService.GetHook(req.Metadata.ProjectID, "onBeforeUpstreamCall"); ok {
		mutated, err := s.jsEngine.ExecuteHook(ctx, script, "onBeforeUpstreamCall", req)
		if err != nil {
			return nil, fmt.Errorf("onBeforeUpstreamCall hook execution failure: %w", err)
		}
		if incoming, validation := mutated.(*domain.InternalRequest); validation {
			req = incoming
		}
	}

	// Execute Actual External Network Operation
	resp, err := providerClient.Call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("upstream destination execution error: %w", err)
	}

	// ----------------------------------------------------
	// HOOK 3: onResponse
	// ----------------------------------------------------
	if script, ok := s.pluginService.GetHook(req.Metadata.ProjectID, "onResponse"); ok {
		mutated, err := s.jsEngine.ExecuteHook(ctx, script, "onResponse", resp)
		if err != nil {
			return nil, fmt.Errorf("onResponse hook execution failure: %w", err)
		}
		if outgoing, validation := mutated.(*domain.InternalResponse); validation {
			resp = outgoing
		}
	}

	// Asynchronously commit usage logs to prevent downstream latency amplification
	go s.budgetService.CommitUsage(context.Background(), req.Metadata, resp.Usage)

	return resp, nil
}

// ExecuteStream pipes real-time streaming chunks back safely without thread blocking.
func (s *GatewayService) ExecuteStream(ctx context.Context, req *domain.InternalRequest) (<-chan domain.StreamChunk, <-chan error) {
	outChunks := make(chan domain.StreamChunk, 100)
	outErr := make(chan error, 1)

	// Execute pre-flight hook adjustments before setting up channels
	if script, ok := s.pluginService.GetHook(req.Metadata.ProjectID, "onRequest"); ok {
		if mutated, err := s.jsEngine.ExecuteHook(ctx, script, "onRequest", req); err == nil {
			if incoming, validation := mutated.(*domain.InternalRequest); validation {
				req = incoming
			}
		}
	}

	if err := s.budgetService.Check(ctx, req.Metadata); err != nil {
		outErr <- err
		close(outChunks)
		close(outErr)
		return outChunks, outErr
	}

	target, err := s.routerService.Route(req)
	if err != nil {
		outErr <- err
		close(outChunks)
		close(outErr)
		return outChunks, outErr
	}

	req.Model = target.TargetModel
	providerClient, exists := s.providers[target.Provider]
	if !exists {
		outErr <- fmt.Errorf("%w: %s", ErrProviderNotFound, target.Provider)
		close(outChunks)
		close(outErr)
		return outChunks, outErr
	}

	// Call underlying streaming capability from the downstream provider
	providerChunks, providerErr := providerClient.StreamCall(ctx, req)

	// Spin asynchronous pipeline processing orchestration
	go func() {
		defer close(outChunks)
		defer close(outErr)

		var fullContent strings.Builder
		var accumulatedUsage domain.TokenUsage

		for {
			select {
			case <-ctx.Done():
				outErr <- ctx.Err()
				return
			case err, ok := <-providerErr:
				if ok && err != nil {
					outErr <- err
					return
				}
			case chunk, ok := <-providerChunks:
				if !ok {
					// End of Stream reached successfully. Calculate fallback usage parameters if required.
					if accumulatedUsage.TotalTokens == 0 {
						accumulatedUsage = s.estimateStreamingTokens(req.Model, fullContent.String())
					}

					// Async tracking persistence
					go s.budgetService.CommitUsage(context.Background(), req.Metadata, accumulatedUsage)
					return
				}

				// Capture generation text updates for background operations
				fullContent.WriteString(chunk.DeltaText)
				if chunk.Usage.TotalTokens > 0 {
					accumulatedUsage = chunk.Usage
				}

				// Apply possible real-time inline chunk modification scripts safely
				if script, ok := s.pluginService.GetHook(req.Metadata.ProjectID, "onResponseChunk"); ok {
					if mutated, err := s.jsEngine.ExecuteHook(ctx, script, "onResponseChunk", chunk); err == nil {
						if modifiedChunk, validation := mutated.(domain.StreamChunk); validation {
							chunk = modifiedChunk
						}
					}
				}

				// Pipe the downstream chunk instantly out to the caller HTTP connection thread
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
