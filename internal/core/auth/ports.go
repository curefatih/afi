package auth

import "context"

type Repository interface {
	SaveAPIKey(ctx context.Context, key *APIKey) error

	GetRequestContextByKeyHash(
		ctx context.Context,
		hash string,
	) (*Principal, error)
}

type KeyGenerator interface {
	Generate(APIKeyType) (string, error)
}

type Hasher interface {
	Hash(string) string
}
