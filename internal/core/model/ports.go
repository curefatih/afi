package model

import "context"

type Repository interface {

	Get(
		ctx context.Context,
		id string,
	) (*Model, error)

	GetByAlias(
		ctx context.Context,
		alias string,
	) (*Model, error)

	ListMappings(
		ctx context.Context,
		modelID string,
	) ([]Mapping, error)
}