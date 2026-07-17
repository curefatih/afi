package auth

type Principal struct {
	OrganizationID string
	TeamID         string
	ProjectID      string

	UserID string

	APIKeyHash string
	APIKeyType APIKeyType
}
