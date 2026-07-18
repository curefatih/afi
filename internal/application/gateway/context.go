package gateway

import (
	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/model"
	"github.com/curefatih/afi/internal/core/pricing"
	"github.com/curefatih/afi/internal/core/provider"
	"github.com/curefatih/afi/internal/core/quota"
	"github.com/curefatih/afi/internal/core/routing"
	"github.com/curefatih/afi/internal/core/usage"
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
	// providerResponses contains every provider interaction during the request.
	// providerResponses []*provider.Response
	// Response remains the final response returned to the client.
	response *provider.Response

	// Quota
	decision *quota.Decision

	// Usage
	usage *usage.Report

	// Pricing
	cost *pricing.Money
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

func (c *Context) Route() *routing.Decision {
	return c.route
}

func (c *Context) SetRoute(r *routing.Decision) {
	c.route = r
}

func (c *Context) Response() *provider.Response {
	return c.response
}

func (c *Context) SetResponse(r *provider.Response) {
	c.response = r
}

func (c *Context) Model() *model.Model {
	return c.model
}

func (c *Context) SetModel(m *model.Model) {
	c.model = m
}

func (c *Context) Usage() *usage.Report {
	return c.usage
}

func (c *Context) SetUsage(u *usage.Report) {
	c.usage = u
}

func (c *Context) Cost() *pricing.Money {
	return c.cost
}

func (c *Context) SetCost(m *pricing.Money) {
	c.cost = m
}
