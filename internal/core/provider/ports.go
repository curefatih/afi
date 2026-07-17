package provider

import "context"

type Client interface {
	Execute(
		context.Context,
		*Request,
	) (*Response, error)
}
