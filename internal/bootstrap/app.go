package bootstrap

import (
	"context"
	"net/http"
)

type App struct {
	Server *http.Server
}

func New(ctx context.Context) (*App, error) {

	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	repositories, err := buildRepositories(cfg)
	if err != nil {
		return nil, err
	}

	services, err := buildServices(repositories)
	if err != nil {
		return nil, err
	}

	transports := buildTransports()
	providers, err := buildProviders(cfg, transports)
	if err != nil {
		return nil, err
	}

	pipeline := buildPipeline(
		services,
		providers,
	)

	server := buildHTTPServer(
		cfg,
		pipeline,
	)

	return &App{
		Server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	return a.Server.ListenAndServe()
}
