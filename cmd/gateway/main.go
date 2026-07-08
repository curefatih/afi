package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/providers"
	"github.com/curefatih/afi/internal/proxy"
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
