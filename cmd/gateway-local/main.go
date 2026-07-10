package main

import (
	"log"
	"net/http"
	"time"

	"github.com/curefatih/afi/internal/core/services"
	"github.com/curefatih/afi/pkg/adapters/inbound/http/openai"
	"github.com/curefatih/afi/pkg/adapters/outbound/jsengine"
	anthropicOutbound "github.com/curefatih/afi/pkg/adapters/outbound/llmproviders/anthropic"
	openaiOutbound "github.com/curefatih/afi/pkg/adapters/outbound/llmproviders/openai"
	"github.com/curefatih/afi/pkg/adapters/outbound/localstatic"

	"github.com/curefatih/afi/internal/ports"
)

func main() {
	log.Println("Initializing standalone local gateway container service runtime...")

	// 1. Instantiate the single YAML static file repository engine
	staticMemoryAdapter, err := localstatic.NewLocalStaticAdapter("configs/local.yaml")
	if err != nil {
		log.Fatalf("Fatal configuration file load failure: %v", err)
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}

	// 2. Wire external outbound providers injecting the common file-backed static vault
	jsSandboxEngine := jsengine.NewGojaEngineAdapter()
	anthropicClient := anthropicOutbound.NewAdapter(staticMemoryAdapter, httpClient)
	openaiClient := openaiOutbound.NewAdapter(staticMemoryAdapter, httpClient)

	providerRegistry := map[string]ports.LLMClient{
		"openai":    openaiClient,
		"anthropic": anthropicClient,
	}

	// 3. Mount services assigning our combined staticMemoryAdapter to all tracking parameters
	authService := services.NewAuthService(staticMemoryAdapter)
	gatewayService := services.NewGatewayService(
		jsSandboxEngine,
		staticMemoryAdapter, // PluginService interface provider
		services.NewBudgetService(staticMemoryAdapter), // BudgetService dependency core
		staticMemoryAdapter,                            // RouterService engine provider
		providerRegistry,
	)

	// 4. Start HTTP Handler setup
	openAIHTTPHandler := openai.NewHandler(gatewayService, authService)
	mux := http.NewServeMux()
	mux.Handle("/v1/chat/completions", openAIHTTPHandler)

	log.Println("🚀 Local Hexagonal Gateway running smoothly on port :8080! Test via cURL targeting key: 'sk-project-local-dev-token-12345'")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
