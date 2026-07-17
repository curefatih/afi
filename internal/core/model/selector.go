package model

import "context"

type Selector interface {
	Resolve(
		ctx context.Context,
		name string,
	) (*Model, error)
}