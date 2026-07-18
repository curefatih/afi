package routing

import (
	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/model"
	"github.com/curefatih/afi/internal/core/provider"
)

type ResolveRequest struct {
	Principal *auth.Principal
	Model     *model.Model
	Request   *provider.Request
	Metadata  map[string]string
}
