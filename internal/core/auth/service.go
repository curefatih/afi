package auth

import "context"

type Service struct {
	repo Repository

	generator KeyGenerator

	hasher Hasher
}

func NewService(repo Repository, generator KeyGenerator, hasher Hasher) *Service {
	return &Service{
		repo:      repo,
		generator: generator,
		hasher:    hasher,
	}
}

func (s *Service) IssueAPIKey(
	ctx context.Context,
	keyType APIKeyType,
	targetID string,
) (string, error) {

	rawKey, err := s.generator.Generate(keyType)
	if err != nil {
		return "", err
	}

	key := &APIKey{
		HashedKey: HashKey(rawKey),
		Type:      keyType,
	}

	switch keyType {

	case KeyTypePersonal:
		key.UserID = targetID

	case KeyTypeServiceAccount:
		key.ProjectID = targetID
	}

	if err := s.repo.SaveAPIKey(ctx, key); err != nil {
		return "", err
	}

	return rawKey, nil
}

func (s *Service) Authenticate(
	ctx context.Context,
	rawKey string,
) (*Principal, error) {

	reqCtx, err := s.repo.GetRequestContextByKeyHash(
		ctx,
		s.hasher.Hash(rawKey),
	)

	if err != nil {
		return nil, ErrUnauthorized
	}

	return reqCtx, nil
}
