package controlplane

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// schemaVersion is the latest schema. Bumps apply additive migrations only.
const schemaVersion = 18

const dropAllSQL = `
DROP TABLE IF EXISTS platform_event_outbox CASCADE;
DROP TABLE IF EXISTS usage_outbox CASCADE;
DROP TABLE IF EXISTS quota_counters CASCADE;
DROP TABLE IF EXISTS quotas CASCADE;
DROP TABLE IF EXISTS request_policies CASCADE;
DROP TABLE IF EXISTS a2a_agents CASCADE;
DROP TABLE IF EXISTS mcp_backends CASCADE;
DROP TABLE IF EXISTS wasm_hooks CASCADE;
DROP TABLE IF EXISTS model_prices CASCADE;
DROP TABLE IF EXISTS usage_events CASCADE;
DROP TABLE IF EXISTS api_key_provider_scopes CASCADE;
DROP TABLE IF EXISTS api_key_providers CASCADE;
DROP TABLE IF EXISTS user_assignments CASCADE;
DROP TABLE IF EXISTS platform_users CASCADE;
DROP TABLE IF EXISTS gateway_snapshots CASCADE;
DROP TABLE IF EXISTS credential_assignments CASCADE;
DROP TABLE IF EXISTS provider_credentials CASCADE;
DROP TABLE IF EXISTS routes CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;
DROP TABLE IF EXISTS providers CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
DROP TABLE IF EXISTS team_members CASCADE;
DROP TABLE IF EXISTS teams CASCADE;
DROP TABLE IF EXISTS organization_invites CASCADE;
DROP TABLE IF EXISTS organization_members CASCADE;
DROP TABLE IF EXISTS external_identities CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;
DROP TABLE IF EXISTS afi_schema_meta CASCADE;
`

const schemaSQL = `
CREATE TABLE IF NOT EXISTS afi_schema_meta (
    version INT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    mail_provider TEXT NOT NULL DEFAULT '',
    byok_selector_header TEXT NOT NULL DEFAULT '',
    default_retry JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'admin',
    password_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS external_identities (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    issuer TEXT NOT NULL DEFAULT '',
    subject TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, subject)
);
CREATE INDEX IF NOT EXISTS external_identities_user_idx
    ON external_identities (user_id);

CREATE TABLE IF NOT EXISTS organization_members (
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member',
    PRIMARY KEY (organization_id, user_id)
);

CREATE TABLE IF NOT EXISTS organization_invites (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    token_hash TEXT NOT NULL UNIQUE,
    invited_by_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS teams (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS team_members (
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member',
    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    team_id TEXT REFERENCES teams(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'service_account',
    owner_user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT api_keys_kind_check CHECK (
        (kind = 'personal' AND owner_user_id IS NOT NULL AND project_id IS NULL) OR
        (kind = 'service_account' AND owner_user_id IS NULL)
    )
);

CREATE TABLE IF NOT EXISTS providers (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    base_url TEXT NOT NULL,
    api_key_env TEXT NOT NULL,
    capabilities JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS provider_credentials (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    provider_type TEXT NOT NULL,
    storage_kind TEXT NOT NULL,
    secret_ref TEXT,
    encrypted_payload BYTEA,
    key_version INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, name),
    CONSTRAINT provider_credentials_storage_check CHECK (
        (storage_kind = 'env' AND secret_ref IS NOT NULL AND encrypted_payload IS NULL) OR
        (storage_kind = 'encrypted_db' AND encrypted_payload IS NOT NULL AND secret_ref IS NULL) OR
        (storage_kind = 'vault' AND secret_ref IS NOT NULL AND encrypted_payload IS NULL)
    ),
    CONSTRAINT provider_credentials_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE IF NOT EXISTS credential_assignments (
    id TEXT PRIMARY KEY,
    credential_id TEXT NOT NULL REFERENCES provider_credentials(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider_type TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    UNIQUE (scope_type, scope_id, provider_type),
    CONSTRAINT credential_assignments_scope_check CHECK (scope_type IN ('organization', 'project', 'api_key'))
);

CREATE TABLE IF NOT EXISTS routes (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    provider_id TEXT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    target_model TEXT NOT NULL,
    fallbacks JSONB NOT NULL DEFAULT '[]'::jsonb,
    retry JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, model)
);

CREATE TABLE IF NOT EXISTS gateway_snapshots (
    version BIGSERIAL PRIMARY KEY,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS usage_events (
    id BIGSERIAL PRIMARY KEY,
    organization_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    api_key_id TEXT NOT NULL DEFAULT '',
    credential_id TEXT,
    used_byok BOOLEAN NOT NULL DEFAULT FALSE,
    model TEXT NOT NULL,
    provider_type TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    prompt_tokens BIGINT NOT NULL DEFAULT 0,
    completion_tokens BIGINT NOT NULL DEFAULT 0,
    cost_usd DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS model_prices (
    provider_type TEXT NOT NULL,
    model TEXT NOT NULL,
    input_per_mtok DOUBLE PRECISION NOT NULL,
    output_per_mtok DOUBLE PRECISION NOT NULL,
    PRIMARY KEY (provider_type, model)
);

CREATE TABLE IF NOT EXISTS quotas (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    metric TEXT NOT NULL,
    limit_value BIGINT NOT NULL,
    time_window TEXT NOT NULL DEFAULT 'total',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quota_counters (
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    metric TEXT NOT NULL,
    time_window TEXT NOT NULL,
    used BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (scope_type, scope_id, metric, time_window)
);

CREATE TABLE IF NOT EXISTS usage_outbox (
    id BIGSERIAL PRIMARY KEY,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS platform_event_outbox (
    id BIGSERIAL PRIMARY KEY,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS request_policies (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    expression TEXT NOT NULL,
    actions JSONB NOT NULL DEFAULT '[]'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    priority INT NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wasm_hooks (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    phase TEXT NOT NULL,
    module_uri TEXT NOT NULL,
    digest TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    priority INT NOT NULL DEFAULT 100,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS wasm_hooks_org_phase_idx
    ON wasm_hooks (organization_id, phase, priority DESC);

CREATE TABLE IF NOT EXISTS mcp_backends (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    alias TEXT NOT NULL,
    name TEXT NOT NULL,
    base_url TEXT NOT NULL,
    api_key_env TEXT NOT NULL DEFAULT '',
    method_allowlist JSONB NOT NULL DEFAULT '[]'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, alias)
);
CREATE INDEX IF NOT EXISTS mcp_backends_org_idx ON mcp_backends (organization_id);

CREATE TABLE IF NOT EXISTS a2a_agents (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    alias TEXT NOT NULL,
    name TEXT NOT NULL,
    upstream_url TEXT NOT NULL,
    card_url TEXT NOT NULL DEFAULT '',
    card_cache JSONB,
    api_key_env TEXT NOT NULL DEFAULT '',
    auth_scheme TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, alias)
);
CREATE INDEX IF NOT EXISTS a2a_agents_org_idx ON a2a_agents (organization_id);
`

// Migrate applies the schema. Legacy UUID installs are wiped once.
// Schema version bumps never drop application data.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	decision, err := inspectSchema(ctx, pool)
	if err != nil {
		return err
	}
	if shouldWipeSchema(decision) {
		if _, err := pool.Exec(ctx, dropAllSQL); err != nil {
			return fmt.Errorf("drop legacy schema: %w", err)
		}
	}

	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err := applyAdditiveMigrations(ctx, pool); err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO afi_schema_meta (version) VALUES ($1)
		ON CONFLICT (version) DO NOTHING
	`, schemaVersion)
	if err != nil {
		return fmt.Errorf("schema meta: %w", err)
	}
	return nil
}

// ResetDatabase drops all AFI tables. Intended for local `afi db reset` only.
func ResetDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, dropAllSQL); err != nil {
		return fmt.Errorf("reset: %w", err)
	}
	return Migrate(ctx, pool)
}

func applyAdditiveMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	var hasKeyValue bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'api_keys' AND column_name = 'key_value'
		)
	`).Scan(&hasKeyValue)
	if err != nil {
		return err
	}
	if hasKeyValue {
		// Cannot recover plaintext→hash; clear keys and move to hash columns (re-seed locally).
		if _, err := pool.Exec(ctx, `
			ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_hash TEXT;
			ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_prefix TEXT NOT NULL DEFAULT '';
			DELETE FROM api_keys;
			ALTER TABLE api_keys DROP COLUMN IF EXISTS key_value;
		`); err != nil {
			return fmt.Errorf("migrate api_keys off key_value: %w", err)
		}
	}

	var hasKeyHash bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'api_keys' AND column_name = 'key_hash'
		)
	`).Scan(&hasKeyHash)
	if err != nil {
		return err
	}
	if hasKeyHash {
		if _, err := pool.Exec(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS api_keys_key_hash_uidx ON api_keys (key_hash);
		`); err != nil {
			return fmt.Errorf("api_keys key_hash index: %w", err)
		}
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS usage_events (
			id BIGSERIAL PRIMARY KEY,
			organization_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			api_key_id TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			status TEXT NOT NULL,
			latency_ms BIGINT NOT NULL DEFAULT 0,
			prompt_tokens BIGINT NOT NULL DEFAULT 0,
			completion_tokens BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS quotas (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			scope_type TEXT NOT NULL,
			scope_id TEXT NOT NULL,
			metric TEXT NOT NULL,
			limit_value BIGINT NOT NULL,
			time_window TEXT NOT NULL DEFAULT 'total',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS quota_counters (
			scope_type TEXT NOT NULL,
			scope_id TEXT NOT NULL,
			metric TEXT NOT NULL,
			time_window TEXT NOT NULL,
			used BIGINT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (scope_type, scope_id, metric, time_window)
		);
		CREATE TABLE IF NOT EXISTS usage_outbox (
			id BIGSERIAL PRIMARY KEY,
			payload JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			processed_at TIMESTAMPTZ
		);
	`); err != nil {
		return fmt.Errorf("cycle3 tables: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE routes ADD COLUMN IF NOT EXISTS fallbacks JSONB NOT NULL DEFAULT '[]'::jsonb;
		ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS cost_usd DOUBLE PRECISION;
		CREATE TABLE IF NOT EXISTS model_prices (
			provider_type TEXT NOT NULL,
			model TEXT NOT NULL,
			input_per_mtok DOUBLE PRECISION NOT NULL,
			output_per_mtok DOUBLE PRECISION NOT NULL,
			PRIMARY KEY (provider_type, model)
		);
		INSERT INTO model_prices (provider_type, model, input_per_mtok, output_per_mtok) VALUES
			('openai', 'gpt-4o-mini', 0.15, 0.60),
			('openai', 'gpt-4o', 2.50, 10.00),
			('anthropic', 'claude-sonnet-4-20250514', 3.00, 15.00),
			('anthropic', 'claude-3-5-sonnet-20241022', 3.00, 15.00),
			('anthropic', 'claude-3-5-haiku-20241022', 0.80, 4.00),
			('gemini', 'gemini-2.0-flash', 0.10, 0.40),
			('gemini', 'gemini-1.5-flash', 0.075, 0.30),
			('gemini', 'gemini-1.5-pro', 1.25, 5.00)
		ON CONFLICT (provider_type, model) DO NOTHING;
	`); err != nil {
		return fmt.Errorf("cycle4 migrations: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE providers ADD COLUMN IF NOT EXISTS capabilities JSONB NOT NULL DEFAULT '{}'::jsonb;
	`); err != nil {
		return fmt.Errorf("cycle5 provider capabilities: %w", err)
	}

	var hasMemberRole bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'organization_members' AND column_name = 'role'
		)
	`).Scan(&hasMemberRole); err != nil {
		return err
	}
	if !hasMemberRole {
		// Existing memberships become owners once; new invites default to member.
		if _, err := pool.Exec(ctx, `
			ALTER TABLE organization_members ADD COLUMN role TEXT NOT NULL DEFAULT 'owner';
			ALTER TABLE organization_members ALTER COLUMN role SET DEFAULT 'member';
		`); err != nil {
			return fmt.Errorf("cycle8 org member role: %w", err)
		}
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'service_account';
		ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS owner_user_id TEXT REFERENCES users(id) ON DELETE CASCADE;
		ALTER TABLE api_keys ALTER COLUMN project_id DROP NOT NULL;
	`); err != nil {
		return fmt.Errorf("cycle8 key kinds: %w", err)
	}
	// Ensure CHECK exists (idempotent drop+add).
	if _, err := pool.Exec(ctx, `
		ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS api_keys_kind_check;
		ALTER TABLE api_keys ADD CONSTRAINT api_keys_kind_check CHECK (
			(kind = 'personal' AND owner_user_id IS NOT NULL AND project_id IS NULL) OR
			(kind = 'service_account' AND owner_user_id IS NULL)
		);
	`); err != nil {
		return fmt.Errorf("cycle8 api_keys check: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS modality TEXT NOT NULL DEFAULT 'chat';
		ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS metrics JSONB NOT NULL DEFAULT '{}'::jsonb;
		CREATE INDEX IF NOT EXISTS usage_events_org_modality_created
			ON usage_events (organization_id, modality, created_at DESC);
	`); err != nil {
		return fmt.Errorf("cycle10 usage modality: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS request_policies (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			expression TEXT NOT NULL,
			actions JSONB NOT NULL DEFAULT '[]'::jsonb,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			priority INT NOT NULL DEFAULT 100,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS request_policies_org_idx
			ON request_policies (organization_id, priority DESC);
	`); err != nil {
		return fmt.Errorf("cycle13 request policies: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS platform_event_outbox (
			id BIGSERIAL PRIMARY KEY,
			payload JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			processed_at TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS platform_event_outbox_pending_idx
			ON platform_event_outbox (id) WHERE processed_at IS NULL;
	`); err != nil {
		return fmt.Errorf("cycle14 platform event outbox: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE organizations ADD COLUMN IF NOT EXISTS mail_provider TEXT NOT NULL DEFAULT '';
		CREATE TABLE IF NOT EXISTS organization_invites (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			email TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			token_hash TEXT NOT NULL UNIQUE,
			invited_by_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			status TEXT NOT NULL DEFAULT 'pending',
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			accepted_at TIMESTAMPTZ
		);
		CREATE UNIQUE INDEX IF NOT EXISTS organization_invites_pending_org_email_uidx
			ON organization_invites (organization_id, email)
			WHERE status = 'pending';
	`); err != nil {
		return fmt.Errorf("cycle15 org invites: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS provider_credentials (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			provider_type TEXT NOT NULL,
			storage_kind TEXT NOT NULL,
			secret_ref TEXT,
			encrypted_payload BYTEA,
			key_version INT NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (organization_id, name),
			CONSTRAINT provider_credentials_storage_check CHECK (
				(storage_kind = 'env' AND secret_ref IS NOT NULL AND encrypted_payload IS NULL) OR
				(storage_kind = 'encrypted_db' AND encrypted_payload IS NOT NULL AND secret_ref IS NULL)
			),
			CONSTRAINT provider_credentials_status_check CHECK (status IN ('active', 'disabled'))
		);
		CREATE TABLE IF NOT EXISTS credential_assignments (
			id TEXT PRIMARY KEY,
			credential_id TEXT NOT NULL REFERENCES provider_credentials(id) ON DELETE CASCADE,
			organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			provider_type TEXT NOT NULL,
			scope_type TEXT NOT NULL,
			scope_id TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_by TEXT,
			UNIQUE (scope_type, scope_id, provider_type),
			CONSTRAINT credential_assignments_scope_check CHECK (scope_type IN ('organization', 'project'))
		);
		CREATE INDEX IF NOT EXISTS provider_credentials_org_idx
			ON provider_credentials (organization_id);
		CREATE INDEX IF NOT EXISTS credential_assignments_org_idx
			ON credential_assignments (organization_id);
	`); err != nil {
		return fmt.Errorf("cycle16 provider credentials: %w", err)
	}

	// Cycle 17: SSO external identities + nullable password for federated-only users.
	if _, err := pool.Exec(ctx, `
		ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
		CREATE TABLE IF NOT EXISTS external_identities (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			provider TEXT NOT NULL,
			issuer TEXT NOT NULL DEFAULT '',
			subject TEXT NOT NULL,
			email TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (provider, subject)
		);
		CREATE INDEX IF NOT EXISTS external_identities_user_idx
			ON external_identities (user_id);
	`); err != nil {
		return fmt.Errorf("cycle17 external identities: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '{}'::jsonb;
	`); err != nil {
		return fmt.Errorf("cycle18 usage tags: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS wasm_hooks (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			phase TEXT NOT NULL,
			module_uri TEXT NOT NULL,
			digest TEXT NOT NULL DEFAULT '',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			priority INT NOT NULL DEFAULT 100,
			config JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS wasm_hooks_org_phase_idx
			ON wasm_hooks (organization_id, phase, priority DESC);
	`); err != nil {
		return fmt.Errorf("cycle19 wasm hooks: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS credential_id TEXT;
		ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS used_byok BOOLEAN NOT NULL DEFAULT FALSE;
		CREATE INDEX IF NOT EXISTS usage_events_org_byok_created
			ON usage_events (organization_id, used_byok, created_at DESC);
	`); err != nil {
		return fmt.Errorf("cycle20 usage byok attribution: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE provider_credentials DROP CONSTRAINT IF EXISTS provider_credentials_storage_check;
		ALTER TABLE provider_credentials ADD CONSTRAINT provider_credentials_storage_check CHECK (
			(storage_kind = 'env' AND secret_ref IS NOT NULL AND encrypted_payload IS NULL) OR
			(storage_kind = 'encrypted_db' AND encrypted_payload IS NOT NULL AND secret_ref IS NULL) OR
			(storage_kind = 'vault' AND secret_ref IS NOT NULL AND encrypted_payload IS NULL)
		);
		ALTER TABLE credential_assignments DROP CONSTRAINT IF EXISTS credential_assignments_scope_check;
		ALTER TABLE credential_assignments ADD CONSTRAINT credential_assignments_scope_check CHECK (
			scope_type IN ('organization', 'project', 'api_key')
		);
	`); err != nil {
		return fmt.Errorf("cycle21 vault credentials and api_key scope: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE organizations ADD COLUMN IF NOT EXISTS byok_selector_header TEXT NOT NULL DEFAULT '';
	`); err != nil {
		return fmt.Errorf("cycle22 byok selector header: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE request_policies ADD COLUMN IF NOT EXISTS credential_name TEXT NOT NULL DEFAULT '';
	`); err != nil {
		return fmt.Errorf("cycle23 policy credential_name: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE request_policies ADD COLUMN IF NOT EXISTS action TEXT NOT NULL DEFAULT 'deny';
		ALTER TABLE request_policies ADD COLUMN IF NOT EXISTS action_config JSONB NOT NULL DEFAULT '{}'::jsonb;
	`); err != nil {
		return fmt.Errorf("cycle24 policy action columns: %w", err)
	}

	// One-time: credential_name → use_credential; legacy allow-gates → deny with inverted WHEN.
	var hasV13 bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM afi_schema_meta WHERE version = 13)`).Scan(&hasV13); err != nil {
		return err
	}
	if !hasV13 {
		if _, err := pool.Exec(ctx, `
			UPDATE request_policies
			SET action = 'use_credential',
			    action_config = jsonb_build_object('credential_name', credential_name)
			WHERE COALESCE(credential_name, '') <> ''
			  AND action = 'deny'
			  AND action_config = '{}'::jsonb;

			UPDATE request_policies
			SET expression = '!(' || expression || ')',
			    action = 'deny',
			    action_config = '{}'::jsonb
			WHERE COALESCE(credential_name, '') = ''
			  AND action = 'deny'
			  AND action_config = '{}'::jsonb;
		`); err != nil {
			return fmt.Errorf("cycle24 policy when/then data migrate: %w", err)
		}
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE request_policies ADD COLUMN IF NOT EXISTS actions JSONB NOT NULL DEFAULT '[]'::jsonb;
	`); err != nil {
		return fmt.Errorf("cycle25 policy actions array: %w", err)
	}

	// One-time: single action/action_config → actions JSON array.
	var hasV14 bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM afi_schema_meta WHERE version = 14)`).Scan(&hasV14); err != nil {
		return err
	}
	if !hasV14 {
		if _, err := pool.Exec(ctx, `
			UPDATE request_policies
			SET actions = jsonb_build_array(
				jsonb_build_object(
					'type', COALESCE(NULLIF(action, ''), 'deny'),
					'config', COALESCE(action_config, '{}'::jsonb)
				)
			)
			WHERE COALESCE(jsonb_array_length(actions), 0) = 0
			  AND EXISTS (
			    SELECT 1 FROM information_schema.columns
			    WHERE table_schema = 'public' AND table_name = 'request_policies' AND column_name = 'action'
			  );
		`); err != nil {
			return fmt.Errorf("cycle25 policy actions data migrate: %w", err)
		}
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE routes ADD COLUMN IF NOT EXISTS retry JSONB;
	`); err != nil {
		return fmt.Errorf("cycle26 route retry: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE organizations ADD COLUMN IF NOT EXISTS default_retry JSONB;
	`); err != nil {
		return fmt.Errorf("cycle27 org default retry: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS mcp_backends (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			alias TEXT NOT NULL,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			api_key_env TEXT NOT NULL DEFAULT '',
			method_allowlist JSONB NOT NULL DEFAULT '[]'::jsonb,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (organization_id, alias)
		);
		CREATE INDEX IF NOT EXISTS mcp_backends_org_idx ON mcp_backends (organization_id);
	`); err != nil {
		return fmt.Errorf("cycle28 mcp backends: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS a2a_agents (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			alias TEXT NOT NULL,
			name TEXT NOT NULL,
			upstream_url TEXT NOT NULL,
			card_url TEXT NOT NULL DEFAULT '',
			card_cache JSONB,
			api_key_env TEXT NOT NULL DEFAULT '',
			auth_scheme TEXT NOT NULL DEFAULT '',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (organization_id, alias)
		);
		CREATE INDEX IF NOT EXISTS a2a_agents_org_idx ON a2a_agents (organization_id);
	`); err != nil {
		return fmt.Errorf("cycle29 a2a agents: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS provider_type TEXT NOT NULL DEFAULT '';
	`); err != nil {
		return fmt.Errorf("cycle30 usage provider_type: %w", err)
	}
	return nil
}

func inspectSchema(ctx context.Context, pool *pgxpool.Pool) (schemaDecision, error) {
	var d schemaDecision
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'afi_schema_meta'
		)
	`).Scan(&d.MetaExists)
	if err != nil {
		return d, err
	}

	if d.MetaExists {
		err := pool.QueryRow(ctx, `SELECT version FROM afi_schema_meta ORDER BY version DESC LIMIT 1`).Scan(&d.MetaVersion)
		if errors.Is(err, pgx.ErrNoRows) {
			d.MetaVersionOK = false
		} else if err != nil {
			return d, err
		} else {
			d.MetaVersionOK = true
		}
	}

	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'organizations'
		)
	`).Scan(&d.OrgExists)
	if err != nil {
		return d, err
	}
	if d.OrgExists {
		_ = pool.QueryRow(ctx, `
			SELECT data_type FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'organizations' AND column_name = 'id'
		`).Scan(&d.OrgIDDataType)
	}
	return d, nil
}
