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
  organization_id UUID NOT NULL REFERENCES organizations(id),
  name VARCHAR(255) NOT NULL DEFAULT 'Unnamed Team',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 3. Projects
CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  team_id UUID NOT NULL REFERENCES teams(id),
  name VARCHAR(255) NOT NULL DEFAULT 'Unnamed Project',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 4. Platform Users
CREATE TABLE IF NOT EXISTS platform_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id),
  name VARCHAR(255) NOT NULL DEFAULT 'Unnamed User',
  email VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  provider VARCHAR(255) NOT NULL DEFAULT 'local',
  external_id VARCHAR(255) NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 5. API Keys
CREATE TABLE IF NOT EXISTS api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id),
  key VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 6. API Key Providers
CREATE TABLE IF NOT EXISTS api_key_providers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  api_key_id UUID NOT NULL REFERENCES api_keys(id),
  provider VARCHAR(255) NOT NULL,
  -- Fixed from 'scope' to 'provider' to match seed data
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 7. API Key Scopes (Renamed the duplicate table to avoid collisions)
CREATE TABLE IF NOT EXISTS api_key_provider_scopes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  api_key_provider_id UUID NOT NULL REFERENCES api_key_providers(id),
  scope VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
---
-- IDEMPOTENT SEED DATA (Using valid UUID v4 formats)
---
-- Seed Organization
INSERT INTO organizations (id, name)
VALUES (
    '00000000-0000-4000-a000-000000000001',
    'Example Organization'
  ) ON CONFLICT (id) DO NOTHING;
-- Seed Team
INSERT INTO teams (id, organization_id, name)
VALUES (
    '00000000-0000-4000-a000-000000000002',
    '00000000-0000-4000-a000-000000000001',
    'Example Team'
  ) ON CONFLICT (id) DO NOTHING;
-- Seed Project
INSERT INTO projects (id, team_id, name)
VALUES (
    '00000000-0000-4000-a000-000000000003',
    '00000000-0000-4000-a000-000000000002',
    'Example Project'
  ) ON CONFLICT (id) DO NOTHING;
-- Seed API Key
INSERT INTO api_keys (id, project_id, key)
VALUES (
    '00000000-0000-4000-a000-000000000004',
    '00000000-0000-4000-a000-000000000003',
    'sk-project-local-dev-token-12345'
  ) ON CONFLICT (id) DO NOTHING;
-- Seed API Key Provider
INSERT INTO api_key_providers (id, api_key_id, provider)
VALUES (
    '00000000-0000-4000-a000-000000000005',
    '00000000-0000-4000-a000-000000000004',
    'openai'
  ) ON CONFLICT (id) DO NOTHING;
-- Seed API Key Provider Scope
INSERT INTO api_key_provider_scopes (id, api_key_provider_id, scope)
VALUES (
    '00000000-0000-4000-a000-000000000006',
    '00000000-0000-4000-a000-000000000005',
    'PROJECT'
  ) ON CONFLICT (id) DO NOTHING;
-- Seed User (Password 'Admin123!')
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