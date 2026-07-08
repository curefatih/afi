package providers

import (
	"github.com/curefatih/afi/internal/config"
)

func BuildRegistry(cfg *config.Config) *Registry {
	items := make(map[string]Provider)
	for name, pcfg := range cfg.Providers {
		switch name {
		default:
			items[name] = NewOpenAI(name, pcfg)
		}
	}
	return NewRegistry(items)
}
