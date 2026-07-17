package auth

type APIKeyType string

const (
	KeyTypePersonal       APIKeyType = "PERSONAL"
	KeyTypeServiceAccount APIKeyType = "SERVICE_ACCOUNT"
)

type APIKey struct {
	ID string

	HashedKey string

	Type APIKeyType

	UserID string

	ProjectID string

	CreatedAt int64
}
