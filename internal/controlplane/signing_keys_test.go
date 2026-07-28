package controlplane

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

type signingKeysFake struct {
	fakePlatform
	keys map[string]*SigningKey
}

func newSigningKeysFake() *signingKeysFake {
	f := &signingKeysFake{keys: map[string]*SigningKey{}}
	f.allowed = map[string]bool{"user_admin|org_a": true, "user_member|org_a": true}
	f.admins = map[string]bool{"user_admin|org_a": true}
	return f
}

func validPublicKeyPEM(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func (f *signingKeysFake) ListSigningKeys(_ context.Context, orgID string) ([]SigningKey, error) {
	var out []SigningKey
	for _, k := range f.keys {
		if k.OrganizationID == orgID {
			out = append(out, *k)
		}
	}
	return out, nil
}

func (f *signingKeysFake) CreateSigningKey(_ context.Context, orgID, keyID, projectID, environmentID, name, algorithm, publicKeyPEM string) (*SigningKey, error) {
	k := &SigningKey{
		ID:             "sig_" + keyID,
		KeyID:          keyID,
		ProjectID:      projectID,
		EnvironmentID:  environmentID,
		OrganizationID: orgID,
		Name:           name,
		Algorithm:      algorithm,
		PublicKeyPEM:   publicKeyPEM,
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	f.keys[k.ID] = k
	return k, nil
}

func (f *signingKeysFake) UpdateSigningKey(_ context.Context, id, name, status string) (*SigningKey, error) {
	k, ok := f.keys[id]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	if name != "" {
		k.Name = name
	}
	if status != "" {
		k.Status = status
	}
	return k, nil
}

func (f *signingKeysFake) RotateSigningKey(_ context.Context, id, publicKeyPEM string) (*SigningKey, error) {
	k, ok := f.keys[id]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	k.PublicKeyPEM = publicKeyPEM
	k.Status = "active"
	return k, nil
}

func (f *signingKeysFake) GetSigningKeyOrgID(_ context.Context, id string) (string, error) {
	k, ok := f.keys[id]
	if !ok {
		return "", kernel.ErrNotFound
	}
	return k.OrganizationID, nil
}

func (f *signingKeysFake) DeleteSigningKey(_ context.Context, id string) error {
	delete(f.keys, id)
	return nil
}

func TestCreateSigningKeyAsAdmin(t *testing.T) {
	api := newSigningKeysFake()
	cfg := testCfg()
	s := &Server{cfg: cfg, api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default()}
	tok := bearer(t, cfg, "user_admin", "admin@afi.local")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_a/signing-keys",
		bytes.NewBufferString(`{"key_id":"svc-a","name":"service a","algorithm":"ed25519","public_key_pem":`+jsonString(validPublicKeyPEM(t))+`}`))
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var k SigningKey
	if err := json.Unmarshal(rr.Body.Bytes(), &k); err != nil {
		t.Fatal(err)
	}
	if k.KeyID != "svc-a" || k.OrganizationID != "org_a" {
		t.Fatalf("%+v", k)
	}
}

func TestUpdateSigningKeyForbiddenForMember(t *testing.T) {
	api := newSigningKeysFake()
	api.keys["sig_ok"] = &SigningKey{ID: "sig_ok", KeyID: "svc-a", OrganizationID: "org_a", Name: "svc", Status: "active"}
	cfg := testCfg()
	s := &Server{cfg: cfg, api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default()}
	tok := bearer(t, cfg, "user_member", "member@afi.local")
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/signing-keys/sig_ok",
		bytes.NewBufferString(`{"status":"disabled"}`))
	req.SetPathValue("signingKeyID", "sig_ok")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func jsonString(v string) string {
	b, _ := json.Marshal(v)
	return string(b)
}
