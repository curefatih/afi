package stages

import (
	"context"

	"github.com/curefatih/afi/internal/application/gateway"
	"github.com/curefatih/afi/internal/core/auth"
)

type Authenticate struct {
	auth auth.Service
}

func NewAuthenticate(authService *auth.Service) *Authenticate {
	return &Authenticate{
		auth: *authService,
	}
}

func (s *Authenticate) Name() string {
	return "authenticate"
}

func (s *Authenticate) Execute(
	ctx context.Context,
	state *gateway.Context,
) error {

	principal, err := s.auth.Authenticate(
		ctx,
		state.Request().APIKey,
	)

	if err != nil {
		return err
	}

	state.SetPrincipal(principal)

	return nil
}
