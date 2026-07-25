package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/modelcatalog"
	"github.com/curefatih/afi/internal/providercatalog"
	"github.com/jackc/pgx/v5/pgconn"
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
	spec, ok := providercatalog.Lookup("echo")
	if !ok {
		return fmt.Errorf("echo provider type not in catalog")
	}
	if err := upsertCatalogProvider(ctx, w.Pool, orgID, now, spec, spec.SeedProviderID(), spec.DefaultBaseURL, spec.DefaultAPIKeyEnv); err != nil {
		return fmt.Errorf("ensure echo provider: %w", err)
	}
	if spec.SeedRoute == nil {
		return nil
	}
	_, err := w.Pool.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (organization_id, model) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			target_model = EXCLUDED.target_model
	`, spec.SeedRoute.ID, orgID, spec.SeedRoute.Model, spec.SeedProviderID(), spec.SeedRoute.TargetModel, now)
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

// EnsureElevenLabs upserts the ElevenLabs provider and curated TTS/STT routes.
func (w *SeedWriter) EnsureElevenLabs(ctx context.Context, orgID string, now time.Time) error {
	return ensureElevenLabsTx(ctx, w.Pool, orgID, now)
}

func ensureElevenLabsTx(ctx context.Context, db seedExecer, orgID string, now time.Time) error {
	spec, ok := providercatalog.Lookup("elevenlabs")
	if !ok {
		return fmt.Errorf("elevenlabs provider type not in catalog")
	}
	providerID := spec.SeedProviderID()
	if err := upsertCatalogProviderTx(ctx, db, orgID, now, spec, providerID, spec.DefaultBaseURL, spec.DefaultAPIKeyEnv); err != nil {
		return fmt.Errorf("ensure elevenlabs provider: %w", err)
	}
	for _, m := range modelcatalog.List("elevenlabs") {
		routeID := "route_" + strings.ReplaceAll(m.ID, "-", "_")
		_, err := db.Exec(ctx, `
			INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
			VALUES ($1, $2, $3, $4, $3, $5)
			ON CONFLICT (organization_id, model) DO UPDATE SET
				provider_id = EXCLUDED.provider_id,
				target_model = EXCLUDED.target_model
		`, routeID, orgID, m.ID, providerID, now)
		if err != nil {
			return fmt.Errorf("ensure elevenlabs route %s: %w", m.ID, err)
		}
	}
	return nil
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

	for _, spec := range providercatalog.Seedable() {
		id := spec.SeedProviderID()
		baseURL := spec.DefaultBaseURL
		apiKeyEnv := spec.DefaultAPIKeyEnv
		if spec.Type == "openai" {
			id = s.ProviderID
			baseURL = s.OpenAIBaseURL
			apiKeyEnv = s.OpenAIAPIKeyEnv
		}
		if err := upsertCatalogProviderTx(ctx, tx, s.OrgID, s.Now, spec, id, baseURL, apiKeyEnv); err != nil {
			return fmt.Errorf("provider %s: %w", spec.Type, err)
		}
		if spec.SeedRoute != nil {
			_, err = tx.Exec(ctx, `
				INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (organization_id, model) DO UPDATE SET
					provider_id = EXCLUDED.provider_id,
					target_model = EXCLUDED.target_model
			`, spec.SeedRoute.ID, s.OrgID, spec.SeedRoute.Model, id, spec.SeedRoute.TargetModel, s.Now)
			if err != nil {
				return fmt.Errorf("%s route: %w", spec.Type, err)
			}
		}
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

	if err := ensureElevenLabsTx(ctx, tx, s.OrgID, s.Now); err != nil {
		return err
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

func capsJSON(c providercatalog.Capabilities) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type seedExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func upsertCatalogProvider(ctx context.Context, db seedExecer, orgID string, now time.Time, spec providercatalog.Spec, id, baseURL, apiKeyEnv string) error {
	return upsertCatalogProviderTx(ctx, db, orgID, now, spec, id, baseURL, apiKeyEnv)
}

func upsertCatalogProviderTx(ctx context.Context, db seedExecer, orgID string, now time.Time, spec providercatalog.Spec, id, baseURL, apiKeyEnv string) error {
	caps, err := capsJSON(spec.Capabilities)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, capabilities, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			capabilities = EXCLUDED.capabilities
	`, id, orgID, spec.DisplayName, spec.Type, baseURL, apiKeyEnv, caps, now)
	return err
}
