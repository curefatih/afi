package dataplane

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dunglas/httpsfv"
	"github.com/yaronf/httpsign"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
	afisign "github.com/curefatih/afi/sdk/httpsign"
)

const (
	headerSignature      = "Signature"
	headerSignatureInput = "Signature-Input"
	headerContentDigest  = "Content-Digest"

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
	return strings.TrimSpace(r.Header.Get(headerSignature)) != "" &&
		strings.TrimSpace(r.Header.Get(headerSignatureInput)) != ""
}

func authenticateGatewayRequest(ctx context.Context, snap *snapshot.Snapshot, replay ReplayStore, r *http.Request, body []byte) (snapshot.Principal, error) {
	if snap == nil {
		return snapshot.Principal{}, kernel.ErrNotFound
	}
	var principal snapshot.Principal
	var err error
	if hasSignedRequestHeaders(r) {
		principal, err = authenticateSignedRequest(ctx, snap, replay, r, body)
	} else {
		var key snapshot.APIKey
		key, err = AuthenticateKey(snap, mustVirtualAPIKey(r))
		if err == nil {
			principal = snapshot.PrincipalFromAPIKey(key)
		}
	}
	if err != nil {
		return snapshot.Principal{}, err
	}
	if !snap.AllowsOrganization(principal.OrganizationID) {
		return snapshot.Principal{}, kernel.ErrForbidden
	}
	return principal, nil
}

func mustVirtualAPIKey(r *http.Request) string {
	raw, _ := virtualAPIKey(r)
	return raw
}

func authenticateSignedRequest(ctx context.Context, snap *snapshot.Snapshot, replay ReplayStore, r *http.Request, body []byte) (snapshot.Principal, error) {
	if body == nil {
		body = []byte{}
	}
	sigName, err := resolveSignatureName(r)
	if err != nil {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	details, err := httpsign.RequestDetails(sigName, r)
	if err != nil || details == nil {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	if details.KeyID == nil || strings.TrimSpace(*details.KeyID) == "" {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	if details.Nonce == nil || strings.TrimSpace(*details.Nonce) == "" {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	keyID := strings.TrimSpace(*details.KeyID)

	signer, ok := snap.LookupSigningKey(keyID)
	if !ok {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	pub, err := access.ParseEd25519PublicKeyPEM(signer.PublicKeyPEM)
	if err != nil {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}

	// Ensure Content-Digest matches the raw body (RFC 9530), independent of req.Body state.
	bodyRC := io.NopCloser(bytes.NewReader(body))
	if err := httpsign.ValidateContentDigestHeader(
		r.Header.Values(headerContentDigest),
		&bodyRC,
		[]string{httpsign.DigestSha256},
	); err != nil {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	// Restore body for signature verification (covers content-digest header value).
	r.Body = io.NopCloser(bytes.NewReader(body))

	verifyCfg := httpsign.NewVerifyConfig().
		SetKeyID(keyID).
		SetAllowedAlgs([]string{"ed25519"}).
		SetVerifyCreated(true).
		SetNotOlderThan(defaultSignedRequestMaxSkew).
		SetNotNewerThan(defaultSignedRequestMaxSkew).
		SetNonceValidator(func(n string) error {
			n = strings.TrimSpace(n)
			if n == "" {
				return fmt.Errorf("empty nonce")
			}
			if replay == nil {
				return nil
			}
			ok, err := replay.Use(ctx, keyID+":"+n, defaultReplayTTL)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("nonce replayed")
			}
			return nil
		})

	verifier, err := httpsign.NewEd25519Verifier(pub, verifyCfg, afisign.RequiredFields())
	if err != nil {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	if err := httpsign.VerifyRequest(sigName, *verifier, r); err != nil {
		return snapshot.Principal{}, kernel.ErrUnauthorized
	}
	return snapshot.PrincipalFromSigningKey(signer), nil
}

func resolveSignatureName(r *http.Request) (string, error) {
	dict, err := httpsfv.UnmarshalDictionary(r.Header.Values(headerSignatureInput))
	if err != nil || dict == nil {
		return "", fmt.Errorf("invalid Signature-Input")
	}
	names := dict.Names()
	if len(names) == 0 {
		return "", fmt.Errorf("empty Signature-Input")
	}
	for _, name := range names {
		if name == afisign.SignatureName {
			return afisign.SignatureName, nil
		}
	}
	return names[0], nil
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
	if errors.Is(err, kernel.ErrForbidden) {
		return "organization not allowed in this region"
	}
	if errors.Is(err, kernel.ErrUnauthorized) {
		return "missing or invalid authorization"
	}
	return fmt.Sprintf("authentication failed: %v", err)
}

func authHTTPStatus(err error) int {
	if errors.Is(err, kernel.ErrNotFound) {
		return http.StatusServiceUnavailable
	}
	if errors.Is(err, kernel.ErrForbidden) {
		return http.StatusForbidden
	}
	return http.StatusUnauthorized
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
