package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/domain"
	"github.com/curefatih/afi/internal/providers"
	"github.com/curefatih/afi/internal/proxy"
	"github.com/curefatih/afi/internal/store"
	"github.com/curefatih/afi/internal/telemetry"
)

func main() {
	configPath := flag.String("config", "configs/example.yaml", "path to gateway config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()

	otelProvider, err := telemetry.Init(ctx, cfg.Telemetry)
	if err != nil {
		slog.Error("init telemetry", "error", err)
		os.Exit(1)
	}
	defer otelProvider.Shutdown(ctx)

	var st domain.Store
	if cfg.Database.URL != "" {
		st = store.NewPostgresStore(cfg.Database.URL, cfg.Database.AutoMigrate, cfg.Database.MigrationsDir)
		if err := st.Open(); err != nil {
			slog.Error("init store", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := st.Close(); err != nil {
				slog.Error("close store", "error", err)
			}
		}()
	}

	hooks, err := proxy.NewHookRunner(cfg)
	if err != nil {
		log.Fatalf("init hooks: %v", err)
	}

	proxyHandler := proxy.NewHandler(proxy.HandlerDeps{
		Config:   cfg,
		Registry: providers.BuildRegistry(cfg),
		Hooks:    hooks,
	})

	apiHandler := http.Handler(proxyHandler)

	mux := http.NewServeMux()
	mux.Handle("/", apiHandler)

	server := proxy.NewServer(cfg.Server.Addr, proxyHandler, cfg.Server.ReadTimeout, cfg.Server.WriteTimeout)
	if err := server.Run(); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}
