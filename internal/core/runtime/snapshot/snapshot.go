package snapshot

import (
	"plugin"

	"github.com/curefatih/afi/internal/core/model"
	"github.com/curefatih/afi/internal/core/provider"
	"github.com/curefatih/afi/internal/core/quota"
	"github.com/curefatih/afi/internal/core/routing"
)

type Snapshot struct {
	Models map[string]*model.Model

	Routes map[string]*routing.Route

	Providers map[string]*provider.Provider

	Quotas []quota.Limit

	Plugins []plugin.Plugin
}
