package gateway

import (
	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/model"
	"github.com/curefatih/afi/internal/core/pricing"
	"github.com/curefatih/afi/internal/core/provider"
	"github.com/curefatih/afi/internal/core/quota"
	"github.com/curefatih/afi/internal/core/routing"
)

type Context struct {
	// Incoming request
	request *provider.Request

	// Authentication
	principal *auth.Principal

	// Resolution
	model *model.Model

	route *routing.Decision

	// Provider
	response *provider.Response

	// Quota

	decision *quota.Decision

	// Pricing

	cost pricing.Money
}

func (c *Context) Request() *provider.Request {
	return c.request
}

func (c *Context) SetPrincipal(p *auth.Principal) {
	c.principal = p
}

func (c *Context) Principal() *auth.Principal {
	return c.principal
}
