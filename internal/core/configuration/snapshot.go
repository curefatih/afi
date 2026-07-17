package configuration

import (
	"plugin"

	"github.com/curefatih/afi/internal/core/model"
	"github.com/curefatih/afi/internal/core/provider"
	"github.com/curefatih/afi/internal/core/quota"
	"github.com/curefatih/afi/internal/core/routing"
)

type Snapshot struct {
	Version int64

	Models []model.Model

	Routes []routing.Route

	Quotas []quota.Limit

	Providers []provider.Provider

	Plugins []plugin.Plugin
}
