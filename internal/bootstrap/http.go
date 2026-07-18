package bootstrap

import (
	"net/http"

	httpapi "github.com/curefatih/afi/internal/adapters/transport/http"
	"github.com/curefatih/afi/internal/application/gateway"
)

func buildHTTPServer(
	cfg Config,
	pipeline *gateway.Pipeline,
) *http.Server {

	handler := httpapi.NewHandler(
		pipeline,
	)

	return &http.Server{
		Addr:    cfg.HTTP.Address,
		Handler: handler,
	}
}
