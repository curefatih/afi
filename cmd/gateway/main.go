// cmd/gateway/main.go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/curefatih/afi/internal/core/services"
	"github.com/curefatih/afi/internal/ports"
	"github.com/curefatih/afi/pkg/adapters/inbound/http/openai"
	"github.com/curefatih/afi/pkg/adapters/outbound/jsengine"
	"github.com/curefatih/afi/pkg/adapters/outbound/llmproviders/anthropic"
	openaiOutbound "github.com/curefatih/afi/pkg/adapters/outbound/llmproviders/openai"
)

func main() {
	// 1. Initialize DB / Infrastructure Outbound Components
	// In production, instantiate your actual database and cache repositories here.
	var vaultAdapter ports.CredentialVault // Your DB secret-store adapter
	var pluginRepo ports.PluginService     // Your Postgres script tracker repo
	var budgetRepo ports.BudgetService     // Your Distributed Redis atomic storage repo
	var routerRepo ports.RouterService     // Your active configuration rule repo
	var authRepo services.AuthRepository   // Your secure hash token lookups repo

	httpClient := &http.Client{Timeout: 90 * time.Second}

	// 2. Assemble Outbound Adapter Implementations
	jsSandboxEngine := jsengine.NewGojaEngineAdapter()
	anthropicClient := anthropic.NewAdapter(vaultAdapter, httpClient)
	openaiClient := openaiOutbound.NewAdapter(vaultAdapter, httpClient)

	providerRegistry := map[string]ports.LLMClient{
		"openai":    openaiClient,
		"anthropic": anthropicClient,
	}

	// 3. Assemble Core Business Logic Services
	authService := services.NewAuthService(authRepo)
	gatewayService := services.NewGatewayService(
		jsSandboxEngine,
		pluginRepo,
		budgetRepo,
		routerRepo,
		providerRegistry,
	)

	// 4. Mount Inbound Driving Adapters
	openAIHTTPHandler := openai.NewHandler(gatewayService, authService)

	mux := http.NewServeMux()
	// Expose our clean proxy route endpoint
	mux.Handle("/v1/chat/completions", openAIHTTPHandler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 6 * time.Minute, // Elevated budget threshold limit to allow lengthy streams
	}

	log.Println("⚡ LLM Gateway Engine online and listening for requests on structural port :8080...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Fatal container engine termination: %v", err)
	}
}
