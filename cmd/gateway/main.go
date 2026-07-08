package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/providers"
	"github.com/curefatih/afi/internal/proxy"
)

func main() {
	configPath := flag.String("config", "configs/example.yaml", "path to gateway config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
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
