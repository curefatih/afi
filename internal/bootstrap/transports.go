package bootstrap

import (
	"net/http"
	"time"

	httptransport "github.com/curefatih/afi/internal/adapters/transport/http"
)

type Transports struct {
	HTTP *httptransport.Client
}

func buildTransports() *Transports {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	return &Transports{
		HTTP: httptransport.New(client),
	}
}