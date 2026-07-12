package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/curefatih/afi/internal/core/config"
	"github.com/curefatih/afi/internal/core/services"
	"github.com/curefatih/afi/pkg/adapters/inbound/http/middleware"
	"github.com/curefatih/afi/pkg/adapters/inbound/http/openai"
	"github.com/curefatih/afi/pkg/adapters/outbound/database"
	"github.com/curefatih/afi/pkg/adapters/outbound/jsengine"
	anthropicOutbound "github.com/curefatih/afi/pkg/adapters/outbound/llmproviders/anthropic"
	openaiOutbound "github.com/curefatih/afi/pkg/adapters/outbound/llmproviders/openai"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/curefatih/afi/internal/ports"
	routerAdapter "github.com/curefatih/afi/pkg/adapters/inbound/http/handlers"
	"github.com/curefatih/afi/pkg/adapters/outbound/crypto"
)

func main() {
	log.Println("Initializing standalone local gateway container service runtime...")

	// 1. Determine config path (e.g., from env or flag)
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/local.yaml"
	}

	// 2. Load Configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 3. Inject only what's necessary into your Adapters
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize token system configuration parameters
	tokenSvc := crypto.NewJWTTokenService(cfg.Auth.TokenSecret, cfg.Auth.Issuer, cfg.Auth.TokenDuration)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.ConnectionString)
	if err != nil {
		log.Fatalf("Unable to establish connect connection pool: %v", err)
	}
	defer pool.Close()

	// Instantiate the multi-interface matching adapter wrapper instance
	dbStore := database.NewPostgresStore(pool)

	// Inject cleanly across core decoupling paths
	authService := services.NewAuthService(dbStore)                       // Consumes AuthRepository
	budgetService := services.NewBudgetService(dbStore)                   // Consumes BudgetRepository
	platformUserSvc := services.NewPlatformUserService(dbStore, tokenSvc) // Consumes PlatformUserRepository
	routerService := services.NewRouterService(dbStore)
	pluginService := services.NewPluginService(dbStore)

	httpClient := &http.Client{Timeout: 60 * time.Second}

	// 2. Wire external outbound providers injecting the common file-backed static vault
	jsSandboxEngine := jsengine.NewGojaEngineAdapter()
	anthropicClient := anthropicOutbound.NewAdapter(dbStore, httpClient)
	openaiClient := openaiOutbound.NewAdapter(dbStore, httpClient)

	providerRegistry := map[string]ports.LLMClient{
		"openai":    openaiClient,
		"anthropic": anthropicClient,
	}

	// 3. Mount services assigning our combined staticMemoryAdapter to all tracking parameters
	gatewayService := services.NewGatewayService(
		jsSandboxEngine,
		pluginService,
		budgetService, // BudgetService dependency core
		routerService, // RouterService engine provider
		providerRegistry,
	)

	// 4. Start HTTP Handler setup
	openAIHTTPHandler := openai.NewHandler(gatewayService, authService)
	mux := http.NewServeMux()
	mux.Handle("/v1/chat/completions", openAIHTTPHandler)

	roleHandler := routerAdapter.NewRoleHandler(platformUserSvc, authService)
	userHandler := routerAdapter.NewUserHandler(platformUserSvc)

	routerAdapter.RegisterPlatformRoutes(mux, tokenSvc, userHandler, roleHandler)
	// 5. Build your global infrastructure middleware chain wrapper
	// CORS acts as the entry guard, converting our *http.ServeMux into an http.Handler
	corsConfig := middleware.DefaultCORSConfig()
	var finalHandler http.Handler = middleware.CORS(corsConfig)(mux)

	// 6. Configure a production-hardened Server to prevent streaming timeout drops
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      finalHandler,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	log.Println(fmt.Sprintf("🚀 AFI Gateway running smoothly on port %d! Test via cURL targeting key: 'sk-project-local-dev-token-12345'", cfg.HTTP.Port))
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
