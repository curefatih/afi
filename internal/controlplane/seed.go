package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Seeder struct {
	pool      *pgxpool.Pool
	store     *Store
	snapStore snapshot.Store
	cfg       *kernel.Config
}

func NewSeeder(pool *pgxpool.Pool, store *Store, snapStore snapshot.Store, cfg *kernel.Config) *Seeder {
	return &Seeder{pool: pool, store: store, snapStore: snapStore, cfg: cfg}
}

// SeedIfEmpty inserts local-dev data when the database has no organizations.
// When the DB already has orgs, it still ensures local audio + echo extension routes (idempotent).
func (s *Seeder) SeedIfEmpty(ctx context.Context) error {
	n, err := s.store.CountOrgs(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return s.Seed(ctx)
	}
	if err := s.EnsureLocalAudioRoutes(ctx); err != nil {
		return err
	}
	return s.EnsureEchoExtension(ctx)
}

// EnsureEchoExtension upserts prov_echo + echo-demo route for org_local and republishes.
func (s *Seeder) EnsureEchoExtension(ctx context.Context) error {
	orgID := "org_local"
	now := time.Now().UTC()
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1)`, orgID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	_, err = s.pool.Exec(ctx, `
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
	_, err = s.pool.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
		VALUES ($1, $2, $3, $4, $3, $5)
		ON CONFLICT (organization_id, model) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			target_model = EXCLUDED.target_model
	`, "route_echo", orgID, "echo-demo", "prov_echo", now)
	if err != nil {
		return fmt.Errorf("ensure echo route: %w", err)
	}
	return s.PublishSnapshot(ctx)
}

// EnsureLocalAudioRoutes upserts tts-1 / whisper-1 → prov_openai for org_local and republishes.
func (s *Seeder) EnsureLocalAudioRoutes(ctx context.Context) error {
	orgID := "org_local"
	providerID := "prov_openai"
	now := time.Now().UTC()
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1)`, orgID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM providers WHERE id=$1)`, providerID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	changed := false
	for _, audio := range []struct{ id, model string }{
		{"route_tts", "tts-1"},
		{"route_whisper", "whisper-1"},
	} {
		var had bool
		_ = s.pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM routes WHERE organization_id=$1 AND model=$2)
		`, orgID, audio.model).Scan(&had)
		_, err := s.pool.Exec(ctx, `
			INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
			VALUES ($1, $2, $3, $4, $3, $5)
			ON CONFLICT (organization_id, model) DO UPDATE SET
				provider_id = EXCLUDED.provider_id,
				target_model = EXCLUDED.target_model
		`, audio.id, orgID, audio.model, providerID, now)
		if err != nil {
			return fmt.Errorf("ensure audio route %s: %w", audio.model, err)
		}
		if !had {
			changed = true
		}
	}
	if !changed {
		// Still republish so capability normalize (tts/stt) reaches the gateway after upgrades.
		return s.PublishSnapshot(ctx)
	}
	return s.PublishSnapshot(ctx)
}

// Seed always inserts (or upserts) the standard local-dev dataset and publishes a snapshot.
func (s *Seeder) Seed(ctx context.Context) error {
	cfg := s.cfg.Seed
	orgID := "org_local"
	teamID := "team_local"
	projectID := "proj_local"
	providerID := "prov_openai"
	userID := "user_admin"
	routeID := "route_default"
	keyID := "key_local"

	hash, err := HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO organizations (id, name, created_at) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name
	`, orgID, "Local Org", now)
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
	`, userID, cfg.AdminEmail, cfg.AdminName, hash, now)
	if err != nil {
		return fmt.Errorf("user: %w", err)
	}

	// Ensure user id is known if conflict updated by email
	_ = tx.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, cfg.AdminEmail).Scan(&userID)

	_, err = tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1, $2, 'owner')
		ON CONFLICT (organization_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, orgID, userID)
	if err != nil {
		return fmt.Errorf("org member: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO teams (id, organization_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
	`, teamID, orgID, "Default Team", now)
	if err != nil {
		return fmt.Errorf("team: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO team_members (team_id, user_id, role) VALUES ($1, $2, 'owner')
		ON CONFLICT DO NOTHING
	`, teamID, userID)
	if err != nil {
		return fmt.Errorf("team member: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO projects (id, organization_id, team_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
	`, projectID, orgID, teamID, "Local Project", now)
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
	`, providerID, orgID, "OpenAI", cfg.OpenAIBaseURL, cfg.OpenAIAPIKeyEnv, now)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}

	// Optional Anthropic provider (no default route — wire via Routing UI).
	_, err = tx.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, created_at)
		VALUES ($1, $2, $3, 'anthropic', $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name
	`, "prov_anthropic", orgID, "Anthropic", "https://api.anthropic.com/v1", "ANTHROPIC_API_KEY", now)
	if err != nil {
		return fmt.Errorf("anthropic provider: %w", err)
	}

	// Optional Gemini provider (no default route — wire via Routing UI).
	_, err = tx.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, created_at)
		VALUES ($1, $2, $3, 'gemini', $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name
	`, "prov_gemini", orgID, "Gemini", "https://generativelanguage.googleapis.com/v1beta", "GEMINI_API_KEY", now)
	if err != nil {
		return fmt.Errorf("gemini provider: %w", err)
	}

	// Optional OpenAI-compatible provider (e.g. local Ollama) — no default route.
	_, err = tx.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, created_at)
		VALUES ($1, $2, $3, 'openai_compatible', $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env,
			name = EXCLUDED.name,
			type = EXCLUDED.type
	`, "prov_ollama", orgID, "Ollama (compatible)", "http://127.0.0.1:11434/v1", "OLLAMA_API_KEY", now)
	if err != nil {
		return fmt.Errorf("ollama provider: %w", err)
	}

	// In-process echo extension (no API key) — local verify / demos.
	_, err = tx.Exec(ctx, `
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
		return fmt.Errorf("echo provider: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
		VALUES ($1, $2, $3, $4, $3, $5)
		ON CONFLICT (organization_id, model) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			target_model = EXCLUDED.target_model
	`, "route_echo", orgID, "echo-demo", "prov_echo", now)
	if err != nil {
		return fmt.Errorf("echo route: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
		VALUES ($1, $2, $3, $4, $3, $5)
		ON CONFLICT (organization_id, model) DO UPDATE SET
			provider_id = EXCLUDED.provider_id,
			target_model = EXCLUDED.target_model
	`, routeID, orgID, cfg.DefaultModel, providerID, now)
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
		`, audio.id, orgID, audio.model, providerID, now)
		if err != nil {
			return fmt.Errorf("audio route %s: %w", audio.model, err)
		}
	}

	keyHash := HashAPIKey(cfg.VirtualAPIKey)
	keyPrefix := KeyPrefix(cfg.VirtualAPIKey)
	_, err = tx.Exec(ctx, `
		INSERT INTO api_keys (id, project_id, organization_id, name, kind, owner_user_id, key_hash, key_prefix, created_at)
		VALUES ($1, $2, $3, $4, 'service_account', NULL, $5, $6, $7)
		ON CONFLICT (key_hash) DO UPDATE SET
			name = EXCLUDED.name,
			key_prefix = EXCLUDED.key_prefix,
			kind = EXCLUDED.kind,
			project_id = EXCLUDED.project_id,
			owner_user_id = EXCLUDED.owner_user_id
	`, keyID, projectID, orgID, "Local Dev Key", keyHash, keyPrefix, now)
	if err != nil {
		return fmt.Errorf("api key: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return s.PublishSnapshot(ctx)
}

func (s *Seeder) PublishSnapshot(ctx context.Context) error {
	src, err := s.store.LoadSnapshotSource(ctx)
	if err != nil {
		return err
	}
	snap := snapshot.Compile(src)
	_, err = s.snapStore.Put(ctx, snap)
	return err
}
