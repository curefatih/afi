package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/curefatih/afi/internal/core/services"
	"github.com/curefatih/afi/internal/ports"
	"github.com/curefatih/afi/pkg/adapters/outbound/llmproviders/anthropic"
	"github.com/curefatih/afi/pkg/adapters/outbound/vault"
)

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	vaultAdapter := vault.NewDatabaseVaultAdapter(db)

	sharedHTTPClient := &http.Client{Timeout: 60 * time.Second}

	anthropicProvider := anthropic.NewAdapter(vaultAdapter, sharedHTTPClient)
	// openaiProvider := openai.NewAdapter(vaultAdapter, sharedHTTPClient)

	// 3. Mount clients into your gateway engine mapping map
	providerCluster := map[string]ports.LLMClient{
		"anthropic": anthropicProvider,
		// "openai":    openaiProvider,
	}

	// 4. Instantiate core domain services with pure interfaces
	_ = services.NewGatewayService(
		nil, // jsEngine
		nil, // pluginService
		nil, // budgetService
		nil, // routerService
		providerCluster,
	)

	// Boot transport listener...
}
