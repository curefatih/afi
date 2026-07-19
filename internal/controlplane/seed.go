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
	pool     *pgxpool.Pool
	store    *Store
	snapStore *snapshot.Store
	cfg      *kernel.Config
}

func NewSeeder(pool *pgxpool.Pool, store *Store, snapStore *snapshot.Store, cfg *kernel.Config) *Seeder {
	return &Seeder{pool: pool, store: store, snapStore: snapStore, cfg: cfg}
}

// SeedIfEmpty inserts local-dev data when the database has no organizations.
func (s *Seeder) SeedIfEmpty(ctx context.Context) error {
	n, err := s.store.CountOrgs(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return s.Seed(ctx)
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
		INSERT INTO organization_members (organization_id, user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
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

	keyHash := HashAPIKey(cfg.VirtualAPIKey)
	keyPrefix := KeyPrefix(cfg.VirtualAPIKey)
	_, err = tx.Exec(ctx, `
		INSERT INTO api_keys (id, project_id, organization_id, name, key_hash, key_prefix, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (key_hash) DO UPDATE SET name = EXCLUDED.name, key_prefix = EXCLUDED.key_prefix
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
