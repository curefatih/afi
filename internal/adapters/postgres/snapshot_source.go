package postgres

import (
	"context"
	"encoding/json"

	"github.com/curefatih/afi/internal/snapshot"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SnapshotSourceLoader loads compile inputs from platform tables.
type SnapshotSourceLoader struct {
	Pool *pgxpool.Pool
}

func NewSnapshotSourceLoader(pool *pgxpool.Pool) *SnapshotSourceLoader {
	return &SnapshotSourceLoader{Pool: pool}
}

func (l *SnapshotSourceLoader) Load(ctx context.Context) (snapshot.Source, error) {
	var src snapshot.Source

	keyRows, err := l.Pool.Query(ctx, `
		SELECT id, key_hash, key_prefix, project_id, organization_id, name, kind, owner_user_id FROM api_keys
	`)
	if err != nil {
		return src, err
	}
	defer keyRows.Close()
	for keyRows.Next() {
		var k snapshot.APIKey
		var projectID, ownerUserID *string
		if err := keyRows.Scan(&k.ID, &k.KeyHash, &k.KeyPrefix, &projectID, &k.OrganizationID, &k.Name, &k.Kind, &ownerUserID); err != nil {
			return src, err
		}
		if projectID != nil {
			k.ProjectID = *projectID
		}
		if ownerUserID != nil {
			k.OwnerUserID = *ownerUserID
		}
		src.APIKeys = append(src.APIKeys, k)
	}
	if err := keyRows.Err(); err != nil {
		return src, err
	}

	provRows, err := l.Pool.Query(ctx, `
		SELECT id, type, base_url, api_key_env, name, capabilities FROM providers
	`)
	if err != nil {
		return src, err
	}
	defer provRows.Close()
	for provRows.Next() {
		var p snapshot.Provider
		var caps []byte
		if err := provRows.Scan(&p.ID, &p.Type, &p.BaseURL, &p.APIKeyEnv, &p.Name, &caps); err != nil {
			return src, err
		}
		p.Capabilities = DecodeCapabilities(p.Type, caps)
		src.Providers = append(src.Providers, p)
	}
	if err := provRows.Err(); err != nil {
		return src, err
	}

	routeRows, err := l.Pool.Query(ctx, `
		SELECT organization_id, model, provider_id, target_model, fallbacks, retry FROM routes
	`)
	if err != nil {
		return src, err
	}
	defer routeRows.Close()
	for routeRows.Next() {
		var r snapshot.Route
		var fb, retryRaw []byte
		if err := routeRows.Scan(&r.OrganizationID, &r.Model, &r.ProviderID, &r.TargetModel, &fb, &retryRaw); err != nil {
			return src, err
		}
		for _, f := range DecodeFallbacks(fb) {
			r.Fallbacks = append(r.Fallbacks, snapshot.RouteTarget{
				ProviderID: f.ProviderID, TargetModel: f.TargetModel,
			})
		}
		if rc := DecodeRetry(retryRaw); rc != nil {
			r.Retry = rc.ToSnapshot()
		}
		src.Routes = append(src.Routes, r)
	}
	if err := routeRows.Err(); err != nil {
		return src, err
	}

	quotaRows, err := l.Pool.Query(ctx, `
		SELECT id, organization_id, scope_type, scope_id, metric, limit_value, time_window FROM quotas
	`)
	if err != nil {
		return src, err
	}
	defer quotaRows.Close()
	for quotaRows.Next() {
		var q snapshot.Quota
		if err := quotaRows.Scan(&q.ID, &q.OrganizationID, &q.ScopeType, &q.ScopeID, &q.Metric, &q.LimitValue, &q.Window); err != nil {
			return src, err
		}
		src.Quotas = append(src.Quotas, q)
	}
	if err := quotaRows.Err(); err != nil {
		return src, err
	}

	polRows, err := l.Pool.Query(ctx, `
		SELECT id, organization_id, name, expression, actions, enabled, priority FROM request_policies
	`)
	if err != nil {
		return src, err
	}
	defer polRows.Close()
	for polRows.Next() {
		var p snapshot.Policy
		var actionsJSON []byte
		if err := polRows.Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Expression, &actionsJSON, &p.Enabled, &p.Priority); err != nil {
			return src, err
		}
		if len(actionsJSON) > 0 {
			var acts []struct {
				Type   string          `json:"type"`
				Config json.RawMessage `json:"config"`
			}
			if err := json.Unmarshal(actionsJSON, &acts); err != nil {
				return src, err
			}
			for _, a := range acts {
				p.Actions = append(p.Actions, snapshot.PolicyAction{Type: a.Type, Config: []byte(a.Config)})
			}
		}
		src.Policies = append(src.Policies, p)
	}
	if err := polRows.Err(); err != nil {
		return src, err
	}

	wasmRows, err := l.Pool.Query(ctx, `
		SELECT id, organization_id, name, phase, module_uri, digest, enabled, priority, config
		FROM wasm_hooks
	`)
	if err != nil {
		return src, err
	}
	defer wasmRows.Close()
	for wasmRows.Next() {
		var h snapshot.WasmHook
		if err := wasmRows.Scan(
			&h.ID, &h.OrganizationID, &h.Name, &h.Phase, &h.ModuleURI, &h.Digest,
			&h.Enabled, &h.Priority, &h.Config,
		); err != nil {
			return src, err
		}
		src.WasmHooks = append(src.WasmHooks, h)
	}
	if err := wasmRows.Err(); err != nil {
		return src, err
	}

	credRows, err := l.Pool.Query(ctx, `
		SELECT id, organization_id, name, provider_type, storage_kind, secret_ref, encrypted_payload, key_version, status
		FROM provider_credentials
	`)
	if err != nil {
		return src, err
	}
	defer credRows.Close()
	for credRows.Next() {
		var c snapshot.Credential
		var secretRef *string
		var payload []byte
		if err := credRows.Scan(&c.ID, &c.OrganizationID, &c.Name, &c.ProviderType, &c.StorageKind, &secretRef, &payload, &c.KeyVersion, &c.Status); err != nil {
			return src, err
		}
		if secretRef != nil {
			c.SecretRef = *secretRef
		}
		c.EncryptedPayload = payload
		src.Credentials = append(src.Credentials, c)
	}
	if err := credRows.Err(); err != nil {
		return src, err
	}

	asgRows, err := l.Pool.Query(ctx, `
		SELECT credential_id, provider_type, scope_type, scope_id FROM credential_assignments
	`)
	if err != nil {
		return src, err
	}
	defer asgRows.Close()
	for asgRows.Next() {
		var a snapshot.CredentialAssignment
		if err := asgRows.Scan(&a.CredentialID, &a.ProviderType, &a.ScopeType, &a.ScopeID); err != nil {
			return src, err
		}
		src.Assignments = append(src.Assignments, a)
	}
	if err := asgRows.Err(); err != nil {
		return src, err
	}

	return src, nil
}
