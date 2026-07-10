package vault

import (
	"context"
	"database/sql"
	"errors"
)

var ErrCredentialNotFound = errors.New("no upstream API key registered for this project/provider tier")

type DatabaseVaultAdapter struct {
	db *sql.DB // or your sqlc wrapper would be injected here
}

func NewDatabaseVaultAdapter(db *sql.DB) *DatabaseVaultAdapter {
	return &DatabaseVaultAdapter{db: db}
}

func (v *DatabaseVaultAdapter) GetProviderKey(ctx context.Context, projectID string, provider string) (string, error) {
	query := `
		SELECT encrypted_api_key FROM project_credentials WHERE project_id = $1 AND provider = $2 AND is_active = true
	`
	var encryptedKey string
	err := v.db.QueryRowContext(ctx, query, projectID, provider).Scan(&encryptedKey)
	if err != nil {
		return "", err
	}
	return encryptedKey, nil
}
