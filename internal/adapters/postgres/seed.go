package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedWriter persists local-dev bootstrap data.
type SeedWriter struct {
	Pool *pgxpool.Pool
}

func NewSeedWriter(pool *pgxpool.Pool) *SeedWriter {
	return &SeedWriter{Pool: pool}
}

func (w *SeedWriter) OrgExists(ctx context.Context, orgID string) (bool, error) {
	var exists bool
	err := w.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1)`, orgID).Scan(&exists)
	return exists, err
}

func (w *SeedWriter) ProviderExists(ctx context.Context, providerID string) (bool, error) {
	var exists bool
	err := w.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM providers WHERE id=$1)`, providerID).Scan(&exists)
	return exists, err
}

// UpsertEchoExtension upserts the echo provider + echo-demo route for an org.
func (w *SeedWriter) UpsertEchoExtension(ctx context.Context, orgID string, now time.Time) error {
	_, err := w.Pool.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, capabilities, created_at)
		VALUES ($1, $2, $3, 'echo', $4, $5, $6::jsonb, $7)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			capabilities = EXCLUDED.capabilities
	`, "prov_echo", orgID, "Echo (extension)", "http://localhost/echo", "ECHO_UNUSED",
		`{"chat":true,"stream":false,"tts":false,"stt":false}`, now)
	if err != nil {
		return fmt.Errorf("ensure echo provider: %w", err)
	}
	_, err = w.Pool.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
		VALUES ($1, $2, $3, $4, $3, $5)
		ON CONFLICT (organization_id, model) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			target_model = EXCLUDED.target_model
	`, "route_echo", orgID, "echo-demo", "prov_echo", now)
	if err != nil {
		return fmt.Errorf("ensure echo route: %w", err)
	}
	return nil
}

// EnsureAudioRoutes upserts tts-1 / whisper-1 routes. changed is true when a route was newly created.
func (w *SeedWriter) EnsureAudioRoutes(ctx context.Context, orgID, providerID string, now time.Time) (changed bool, err error) {
	for _, audio := range []struct{ id, model string }{
		{"route_tts", "tts-1"},
		{"route_whisper", "whisper-1"},
	} {
		var had bool
		_ = w.Pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM routes WHERE organization_id=$1 AND model=$2)
		`, orgID, audio.model).Scan(&had)
		_, err := w.Pool.Exec(ctx, `
			INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
			VALUES ($1, $2, $3, $4, $3, $5)
			ON CONFLICT (organization_id, model) DO UPDATE SET
				provider_id = EXCLUDED.provider_id,
				target_model = EXCLUDED.target_model
		`, audio.id, orgID, audio.model, providerID, now)
		if err != nil {
			return false, fmt.Errorf("ensure audio route %s: %w", audio.model, err)
		}
		if !had {
			changed = true
		}
	}
	return changed, nil
}

// LocalDevSeed holds values for the standard local-dev dataset.
type LocalDevSeed struct {
	OrgID           string
	TeamID          string
	ProjectID       string
	ProviderID      string
	UserID          string
	RouteID         string
	KeyID           string
	AdminEmail      string
	AdminName       string
	PasswordHash    string
	OpenAIBaseURL   string
	OpenAIAPIKeyEnv string
	DefaultModel    string
	APIKeyHash      string
	APIKeyPrefix    string
	Now             time.Time
}

// SeedLocalDev upserts the standard local-dev dataset in a transaction.
func (w *SeedWriter) SeedLocalDev(ctx context.Context, s LocalDevSeed) error {
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO organizations (id, name, created_at) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name
	`, s.OrgID, "Local Org", s.Now)
	if err != nil {
		return fmt.Errorf("org: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, name, role, password_hash, created_at)
		VALUES ($1, $2, $3, 'admin', $4, $5)
		ON CONFLICT (email) DO UPDATE SET
			name = EXCLUDED.name,
			password_hash = EXCLUDED.password_hash,
			role = EXCLUDED.role
	`, s.UserID, s.AdminEmail, s.AdminName, s.PasswordHash, s.Now)
	if err != nil {
		return fmt.Errorf("user: %w", err)
	}

	userID := s.UserID
	_ = tx.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, s.AdminEmail).Scan(&userID)

	_, err = tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1, $2, 'owner')
		ON CONFLICT (organization_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, s.OrgID, userID)
	if err != nil {
		return fmt.Errorf("org member: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO teams (id, organization_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
	`, s.TeamID, s.OrgID, "Default Team", s.Now)
	if err != nil {
		return fmt.Errorf("team: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO team_members (team_id, user_id, role) VALUES ($1, $2, 'owner')
		ON CONFLICT DO NOTHING
	`, s.TeamID, userID)
	if err != nil {
		return fmt.Errorf("team member: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO projects (id, organization_id, team_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
	`, s.ProjectID, s.OrgID, s.TeamID, "Local Project", s.Now)
	if err != nil {
		return fmt.Errorf("project: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, created_at)
		VALUES ($1, $2, $3, 'openai', $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name
	`, s.ProviderID, s.OrgID, "OpenAI", s.OpenAIBaseURL, s.OpenAIAPIKeyEnv, s.Now)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, created_at)
		VALUES ($1, $2, $3, 'anthropic', $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name
	`, "prov_anthropic", s.OrgID, "Anthropic", "https://api.anthropic.com/v1", "ANTHROPIC_API_KEY", s.Now)
	if err != nil {
		return fmt.Errorf("anthropic provider: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, created_at)
		VALUES ($1, $2, $3, 'gemini', $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name
	`, "prov_gemini", s.OrgID, "Gemini", "https://generativelanguage.googleapis.com/v1beta", "GEMINI_API_KEY", s.Now)
	if err != nil {
		return fmt.Errorf("gemini provider: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, created_at)
		VALUES ($1, $2, $3, 'openai_compatible', $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name,
			type = EXCLUDED.type
	`, "prov_ollama", s.OrgID, "Ollama (compatible)", "http://127.0.0.1:11434/v1", "OLLAMA_API_KEY", s.Now)
	if err != nil {
		return fmt.Errorf("ollama provider: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, capabilities, created_at)
		VALUES ($1, $2, $3, 'echo', $4, $5, $6::jsonb, $7)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			capabilities = EXCLUDED.capabilities
	`, "prov_echo", s.OrgID, "Echo (extension)", "http://localhost/echo", "ECHO_UNUSED",
		`{"chat":true,"stream":false,"tts":false,"stt":false}`, s.Now)
	if err != nil {
		return fmt.Errorf("echo provider: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
		VALUES ($1, $2, $3, $4, $3, $5)
		ON CONFLICT (organization_id, model) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			target_model = EXCLUDED.target_model
	`, "route_echo", s.OrgID, "echo-demo", "prov_echo", s.Now)
	if err != nil {
		return fmt.Errorf("echo route: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
		VALUES ($1, $2, $3, $4, $3, $5)
		ON CONFLICT (organization_id, model) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			target_model = EXCLUDED.target_model
	`, s.RouteID, s.OrgID, s.DefaultModel, s.ProviderID, s.Now)
	if err != nil {
		return fmt.Errorf("route: %w", err)
	}

	for _, audio := range []struct{ id, model string }{
		{"route_tts", "tts-1"},
		{"route_whisper", "whisper-1"},
	} {
		_, err = tx.Exec(ctx, `
			INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
			VALUES ($1, $2, $3, $4, $3, $5)
			ON CONFLICT (organization_id, model) DO UPDATE SET
				provider_id = EXCLUDED.provider_id,
				target_model = EXCLUDED.target_model
		`, audio.id, s.OrgID, audio.model, s.ProviderID, s.Now)
		if err != nil {
			return fmt.Errorf("audio route %s: %w", audio.model, err)
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO api_keys (id, project_id, organization_id, name, kind, owner_user_id, key_hash, key_prefix, created_at)
		VALUES ($1, $2, $3, $4, 'service_account', NULL, $5, $6, $7)
		ON CONFLICT (key_hash) DO UPDATE SET
			name = EXCLUDED.name,
			key_prefix = EXCLUDED.key_prefix,
			kind = EXCLUDED.kind,
			project_id = EXCLUDED.project_id,
			owner_user_id = EXCLUDED.owner_user_id
	`, s.KeyID, s.ProjectID, s.OrgID, "Local Dev Key", s.APIKeyHash, s.APIKeyPrefix, s.Now)
	if err != nil {
		return fmt.Errorf("api key: %w", err)
	}

	return tx.Commit(ctx)
}
