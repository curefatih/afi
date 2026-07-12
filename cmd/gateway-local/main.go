package main

import (
	"context"
	"log"
	"net/http"
	"time"

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

	// Initialize token system configuration parameters
	jwtSecret := "super-secure-dev-platform-secret-key-change-in-prod"
	tokenDuration := 12 * time.Hour

	tokenSvc := crypto.NewJWTTokenService(jwtSecret, "afi-gateway-platform", tokenDuration)

	ctx := context.Background()
	// TODO: Use environment variables to set the connection string. These are mock
	connStr := "postgresql://postgres:secret@localhost:5432/afi_gateway?sslmode=disable"

	pool, err := pgxpool.New(ctx, connStr)
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
		Addr:         ":8080",
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // 🚨 CRITICAL: Keep 0 to prevent cutting off long-running SSE chunk responses!
	}

	log.Println("🚀 Local Hexagonal Gateway running smoothly on port :8080! Test via cURL targeting key: 'sk-project-local-dev-token-12345'")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
