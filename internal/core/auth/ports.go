package auth

import "context"

type Repository interface {
	SaveAPIKey(ctx context.Context, key *APIKey) error

	GetRequestContextByKeyHash(
		ctx context.Context,
		hash string,
	) (*Principal, error)
}
