package domain

type APIKeyType string

const (
	KeyTypePersonal       APIKeyType = "PERSONAL"
	KeyTypeServiceAccount APIKeyType = "SERVICE_ACCOUNT"
)

type APIKey struct {
	HashedKey string
	Type      APIKeyType
	UserID    string // Set if Personal
	ProjectID string // Set if Service Account (inherits TeamID/OrgID)
}
