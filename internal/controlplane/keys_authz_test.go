package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

type keysFake struct {
	fakePlatform
	keys map[string]*APIKey
}

func newKeysFake() *keysFake {
	f := &keysFake{keys: map[string]*APIKey{}}
	f.allowed = map[string]bool{
		"user_admin|org_a":  true,
		"user_member|org_a": true,
	}
	f.admins = map[string]bool{
		"user_admin|org_a": true,
	}
	return f
}

func (f *keysFake) ListOrgAPIKeys(_ context.Context, orgID string) ([]APIKey, error) {
	var out []APIKey
	for _, k := range f.keys {
		if k.OrganizationID == orgID {
			out = append(out, *k)
		}
	}
	return out, nil
}

func (f *keysFake) GetAPIKey(_ context.Context, keyID string) (*APIKey, error) {
	k, ok := f.keys[keyID]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *k
	return &cp, nil
}

func (f *keysFake) CreateAPIKey(_ context.Context, orgID, kind, ownerUserID, projectID, name, rawKey string) (*APIKey, error) {
	k := &APIKey{
		ID: "key_" + name, OrganizationID: orgID, Kind: kind, OwnerUserID: ownerUserID,
		ProjectID: projectID, Name: name, KeyPrefix: KeyPrefix(rawKey), Key: rawKey,
		CreatedAt: time.Now().UTC(),
	}
	f.keys[k.ID] = k
	return k, nil
}

func (f *keysFake) DeleteAPIKey(_ context.Context, keyID string) error {
	if _, ok := f.keys[keyID]; !ok {
		return kernel.ErrNotFound
	}
	delete(f.keys, keyID)
	return nil
}

func (f *keysFake) CreateQuota(_ context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*Quota, error) {
	if scopeType == snapshot.ScopeUser && scopeID == "missing" {
		return nil, kernel.ErrInvalidRequest
	}
	return &Quota{
		ID: "quota_1", OrganizationID: orgID, ScopeType: scopeType, ScopeID: scopeID,
		Metric: metric, LimitValue: limitValue, Window: window,
	}, nil
}

func (f *keysFake) IsOrgMember(ctx context.Context, userID, orgID string) (bool, error) {
	return f.fakePlatform.IsOrgMember(ctx, userID, orgID)
}

func bearer(t *testing.T, cfg *kernel.Config, userID, email string) string {
	t.Helper()
	tok, err := IssueToken(cfg.Auth.JWTSecret, time.Hour, userID, email, "member")
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestCreatePersonalKeyAsMember(t *testing.T) {
	t.Parallel()
	api := newKeysFake()
	pub := &fakePublisher{}
	cfg := testCfg()
	s := &Server{cfg: cfg, api: api, config: api, members: api, publisher: pub, log: slog.Default()}
	tok := bearer(t, cfg, "user_member", "member@afi.local")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_a/keys",
		bytes.NewBufferString(`{"name":"mine","kind":"personal"}`))
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var k APIKey
	if err := json.Unmarshal(rr.Body.Bytes(), &k); err != nil {
		t.Fatal(err)
	}
	if k.Kind != snapshot.KeyKindPersonal || k.OwnerUserID != "user_member" || k.ProjectID != "" {
		t.Fatalf("%+v", k)
	}
}

func TestCreateServiceAccountForbiddenForMember(t *testing.T) {
	t.Parallel()
	api := newKeysFake()
	cfg := testCfg()
	s := &Server{cfg: cfg, api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default()}
	tok := bearer(t, cfg, "user_member", "member@afi.local")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_a/keys",
		bytes.NewBufferString(`{"name":"bot","kind":"service_account"}`))
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateServiceAccountAsAdmin(t *testing.T) {
	t.Parallel()
	api := newKeysFake()
	cfg := testCfg()
	s := &Server{cfg: cfg, api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default()}
	tok := bearer(t, cfg, "user_admin", "admin@afi.local")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_a/keys",
		bytes.NewBufferString(`{"name":"bot","kind":"service_account"}`))
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDeletePersonalKeyAsOwner(t *testing.T) {
	t.Parallel()
	api := newKeysFake()
	api.keys["key_mine"] = &APIKey{
		ID: "key_mine", OrganizationID: "org_a", Kind: snapshot.KeyKindPersonal,
		OwnerUserID: "user_member", Name: "mine",
	}
	cfg := testCfg()
	s := &Server{cfg: cfg, api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default()}
	tok := bearer(t, cfg, "user_member", "member@afi.local")
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/platform/keys/key_mine", nil)
	req.SetPathValue("keyID", "key_mine")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateQuotaForbiddenForMember(t *testing.T) {
	t.Parallel()
	api := newKeysFake()
	cfg := testCfg()
	s := &Server{cfg: cfg, api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default()}
	tok := bearer(t, cfg, "user_member", "member@afi.local")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_a/quotas",
		bytes.NewBufferString(`{"scope_type":"user","scope_id":"user_member","metric":"requests","limit_value":10}`))
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateUserQuotaAsAdmin(t *testing.T) {
	t.Parallel()
	api := newKeysFake()
	cfg := testCfg()
	s := &Server{cfg: cfg, api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default()}
	tok := bearer(t, cfg, "user_admin", "admin@afi.local")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_a/quotas",
		bytes.NewBufferString(`{"scope_type":"user","scope_id":"user_member","metric":"requests","limit_value":10}`))
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
