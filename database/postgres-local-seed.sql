-- ==========================================
-- MIGRATION: UP (Forward Migration)
-- ==========================================
-- 0. Helper Functions (For automatic updated_at handling)
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- 1. Organizations
CREATE TABLE IF NOT EXISTS organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL DEFAULT 'Unnamed Organization',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 2. Teams
CREATE TABLE IF NOT EXISTS teams (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL DEFAULT 'Unnamed Team',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 3. Projects
CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL DEFAULT 'Unnamed Project',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 4. Platform Users
CREATE TABLE IF NOT EXISTS platform_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL DEFAULT 'Unnamed User',
  email VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  provider VARCHAR(255) NOT NULL DEFAULT 'local',
  external_id VARCHAR(255) NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uq_platform_users_email UNIQUE (email)
);
-- 5. API Keys
CREATE TABLE IF NOT EXISTS api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  key VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 6. API Key Providers
CREATE TABLE IF NOT EXISTS api_key_providers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
  provider VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 7. API Key Scopes
CREATE TABLE IF NOT EXISTS api_key_provider_scopes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  api_key_provider_id UUID NOT NULL REFERENCES api_key_providers(id) ON DELETE CASCADE,
  scope VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 8. User assignments
CREATE TABLE IF NOT EXISTS user_assignments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
  role_name VARCHAR(100) NOT NULL
);
-- Unique index to handle NULL in project_id safely
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_assignments ON user_assignments (
  user_id,
  organization_id,
  COALESCE(
    project_id,
    '00000000-0000-0000-0000-000000000000'
  )
);
-- Idempotent Trigger Registration Block
DO $$ BEGIN -- 1. Organizations Trigger
IF NOT EXISTS (
  SELECT 1
  FROM pg_trigger
  WHERE tgname = 'set_timestamp_organizations'
) THEN CREATE TRIGGER set_timestamp_organizations BEFORE
UPDATE ON organizations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
END IF;
-- 2. Teams Trigger
IF NOT EXISTS (
  SELECT 1
  FROM pg_trigger
  WHERE tgname = 'set_timestamp_teams'
) THEN CREATE TRIGGER set_timestamp_teams BEFORE
UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
END IF;
-- 3. Projects Trigger
IF NOT EXISTS (
  SELECT 1
  FROM pg_trigger
  WHERE tgname = 'set_timestamp_projects'
) THEN CREATE TRIGGER set_timestamp_projects BEFORE
UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
END IF;
-- 4. Platform Users Trigger
IF NOT EXISTS (
  SELECT 1
  FROM pg_trigger
  WHERE tgname = 'set_timestamp_platform_users'
) THEN CREATE TRIGGER set_timestamp_platform_users BEFORE
UPDATE ON platform_users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
END IF;
-- 5. API Keys Trigger
IF NOT EXISTS (
  SELECT 1
  FROM pg_trigger
  WHERE tgname = 'set_timestamp_api_keys'
) THEN CREATE TRIGGER set_timestamp_api_keys BEFORE
UPDATE ON api_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
END IF;
-- 6. API Key Providers Trigger
IF NOT EXISTS (
  SELECT 1
  FROM pg_trigger
  WHERE tgname = 'set_timestamp_api_key_providers'
) THEN CREATE TRIGGER set_timestamp_api_key_providers BEFORE
UPDATE ON api_key_providers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
END IF;
-- 7. API Key Scopes Trigger
IF NOT EXISTS (
  SELECT 1
  FROM pg_trigger
  WHERE tgname = 'set_timestamp_api_key_provider_scopes'
) THEN CREATE TRIGGER set_timestamp_api_key_provider_scopes BEFORE
UPDATE ON api_key_provider_scopes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
END IF;
END $$;
---
-- SEED DATA (Idempotent Execution)
---
INSERT INTO organizations (id, name)
VALUES (
    '00000000-0000-4000-a000-000000000001',
    'Example Organization'
  ) ON CONFLICT (id) DO NOTHING;
INSERT INTO teams (id, organization_id, name)
VALUES (
    '00000000-0000-4000-a000-000000000002',
    '00000000-0000-4000-a000-000000000001',
    'Example Team'
  ) ON CONFLICT (id) DO NOTHING;
INSERT INTO projects (id, team_id, name)
VALUES (
    '00000000-0000-4000-a000-000000000003',
    '00000000-0000-4000-a000-000000000002',
    'Example Project'
  ) ON CONFLICT (id) DO NOTHING;
INSERT INTO api_keys (id, project_id, key)
VALUES (
    '00000000-0000-4000-a000-000000000004',
    '00000000-0000-4000-a000-000000000003',
    'sk-project-local-dev-token-12345'
  ) ON CONFLICT (id) DO NOTHING;
INSERT INTO api_key_providers (id, api_key_id, provider)
VALUES (
    '00000000-0000-4000-a000-000000000005',
    '00000000-0000-4000-a000-000000000004',
    'openai'
  ) ON CONFLICT (id) DO NOTHING;
INSERT INTO api_key_provider_scopes (id, api_key_provider_id, scope)
VALUES (
    '00000000-0000-4000-a000-000000000006',
    '00000000-0000-4000-a000-000000000005',
    'PROJECT'
  ) ON CONFLICT (id) DO NOTHING;
INSERT INTO platform_users (
    id,
    organization_id,
    name,
    email,
    password_hash,
    provider,
    external_id,
    is_active
  )
VALUES (
    '00000000-0000-4000-a000-000000000007',
    '00000000-0000-4000-a000-000000000001',
    'Example User',
    'example@example.com',
    '$2a$10$W27D5MJXibYeJAngYXTXm.KkCWzUKpRj9jD5a9DdVtQV1Bdt9dzIC',
    'local',
    '',
    TRUE
  ) ON CONFLICT (id) DO NOTHING;
INSERT INTO user_assignments (
    user_id,
    organization_id,
    team_id,
    project_id,
    role_name
  )
VALUES (
    '00000000-0000-4000-a000-000000000007',
    '00000000-0000-4000-a000-000000000001',
    '00000000-0000-4000-a000-000000000002',
    '00000000-0000-4000-a000-000000000003',
    'MEMBER'
  ) ON CONFLICT (
    user_id,
    organization_id,
    COALESCE(
      project_id,
      '00000000-0000-0000-0000-000000000000'
    )
  ) DO NOTHING;
-- ==========================================
-- MIGRATION: DOWN (Rollback Migration)
-- ==========================================
/* -- To rollback, uncomment and execute the lines below:
 
 DROP TABLE IF EXISTS user_assignments CASCADE;
 DROP TABLE IF EXISTS api_key_provider_scopes CASCADE;
 DROP TABLE IF EXISTS api_key_providers CASCADE;
 DROP TABLE IF EXISTS api_keys CASCADE;
 DROP TABLE IF EXISTS platform_users CASCADE;
 DROP TABLE IF EXISTS projects CASCADE;
 DROP TABLE IF EXISTS teams CASCADE;
 DROP TABLE IF EXISTS organizations CASCADE;
 DROP FUNCTION IF EXISTS update_updated_at_column();
 */