package dataplane

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

const (
	headerSigningKeyID   = "X-AFI-Key-Id"
	headerSigningTS      = "X-AFI-Timestamp"
	headerSigningNonce   = "X-AFI-Nonce"
	headerSigningBodySHA = "X-AFI-Content-SHA256"
	headerSigningSig     = "X-AFI-Signature"

	defaultSignedRequestMaxSkew = 5 * time.Minute
	defaultReplayTTL            = 10 * time.Minute
)

type ReplayStore interface {
	Use(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

type memoryReplayStore struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func newMemoryReplayStore() *memoryReplayStore {
	return &memoryReplayStore{items: map[string]time.Time{}}
}

func (s *memoryReplayStore) Use(_ context.Context, key string, ttl time.Duration) (bool, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, exp := range s.items {
		if !exp.After(now) {
			delete(s.items, k)
		}
	}
	if exp, ok := s.items[key]; ok && exp.After(now) {
		return false, nil
	}
	s.items[key] = now.Add(ttl)
	return true, nil
}

func hasSignedRequestHeaders(r *http.Request) bool {
	for _, v := range []string{
		r.Header.Get(headerSigningKeyID),
		r.Header.Get(headerSigningTS),
		r.Header.Get(headerSigningNonce),
		r.Header.Get(headerSigningBodySHA),
		r.Header.Get(headerSigningSig),
	} {
		if strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

func authenticateGatewayRequest(ctx context.Context, snap *snapshot.Snapshot, replay ReplayStore, r *http.Request, body []byte) (snapshot.Principal, error) {
	if snap == nil {
		return snapshot.Principal{}, kernel.ErrNotFound
	}
	if hasSignedRequestHeaders(r) {
		return authenticateSignedRequest(ctx, snap, replay, r, body)
	}
	key, err := AuthenticateKey(snap, mustVirtualAPIKey(r))
	if err != nil {
		return snapshot.Principal{}, err
	}
	return snapshot.PrincipalFromAPIKey(key), nil
}

func mustVirtualAPIKey(r *http.Request) string {
	raw, _ := virtualAPIKey(r)
	return raw
}

func authenticateSignedRequest(ctx context.Context, snap *snapshot.Snapshot, replay ReplayStore, r *http.Request, body []byte) (snapshot.Principal, error) {
	keyID := strings.TrimSpace(r.Header.Get(headerSigningKeyID))
	tsRaw := strings.TrimSpace(r.Header.Get(headerSigningTS))
	nonce := strings.TrimSpace(r.Header.Get(headerSigningNonce))
	bodyHash := strings.TrimSpace(r.Header.Get(headerSigningBodySHA))
	sigRaw := strings.TrimSpace(r.Header.Get(headerSigningSig))
	if keyID == "" || tsRaw == "" || nonce == "" || bodyHash == "" || sigRaw == "" {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	signer, ok := snap.LookupSigningKey(keyID)
	if !ok {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	ts, err := time.Parse(time.RFC3339, tsRaw)
	if err != nil {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	now := time.Now().UTC()
	if ts.Before(now.Add(-defaultSignedRequestMaxSkew)) || ts.After(now.Add(defaultSignedRequestMaxSkew)) {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != strings.ToLower(bodyHash) {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	if replay != nil {
		ok, err := replay.Use(ctx, keyID+":"+nonce, defaultReplayTTL)
		if err != nil || !ok {
			if err != nil {
				return snapshot.Principal{}, err
			}
			return snapshot.Principal{}, kernel.ErrUnauthorized
		}
	}
	msg := canonicalSignedRequest(r, bodyHash, tsRaw, nonce)
	sig, err := base64.StdEncoding.DecodeString(sigRaw)
	if err != nil {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	pub, err := access.ParseEd25519PublicKeyPEM(signer.PublicKeyPEM)
	if err != nil {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	if !ed25519.Verify(pub, []byte(msg), sig) {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	return snapshot.PrincipalFromSigningKey(signer), nil
}

func canonicalSignedRequest(r *http.Request, bodyHash, tsRaw, nonce string) string {
	path := r.URL.EscapedPath()
	if q := r.URL.RawQuery; q != "" {
		path += "?" + q
	}
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	return strings.Join([]string{
		strings.ToUpper(r.Method),
		path,
		contentType,
		strings.ToLower(bodyHash),
		tsRaw,
		nonce,
	}, "\n")
}

// AuthenticateKey is exported for unit tests.
func AuthenticateKey(snap *snapshot.Snapshot, rawKey string) (snapshot.APIKey, error) {
	if snap == nil {
		return snapshot.APIKey{}, kernel.ErrNotFound
	}
	if strings.TrimSpace(rawKey) == "" {
		return snapshot.APIKey{}, kernel.ErrUnauthorized
	}
	k, ok := snap.LookupKey(rawKey)
	if !ok {
		return snapshot.APIKey{}, kernel.ErrUnauthorized
	}
	return k, nil
}

func authErrMessage(err error) string {
	if errors.Is(err, kernel.ErrNotFound) {
		return "no snapshot loaded"
	}
	if errors.Is(err, kernel.ErrUnauthorized) {
		return "missing or invalid authorization"
	}
	return fmt.Sprintf("authentication failed: %v", err)
}

func usageEventBase(principal snapshot.Principal, credentialID string) UsageEvent {
	return UsageEvent{
		OrganizationID: principal.OrganizationID,
		ProjectID:      principal.ProjectID,
		TeamID:         principal.TeamID,
		EnvironmentID:  principal.EnvironmentID,
		APIKeyID:       principal.APIKeyID,
		SigningKeyID:   principal.SigningKeyID,
		SignerKeyID:    principal.KeyID,
		AuthMethod:     principal.AuthMethod,
		CredentialID:   credentialID,
	}
}
