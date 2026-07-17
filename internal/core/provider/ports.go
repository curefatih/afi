package provider

import "context"

type Client interface {
	Execute(
		ctx context.Context,
		request *Request,
	) (*Response, error)
}
