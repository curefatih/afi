package dataplane

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yaronf/httpsign"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
	afisign "github.com/curefatih/afi/sdk/httpsign"
)

func testEd25519PublicKeyPEM(t *testing.T) (ed25519.PrivateKey, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return priv, string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func signGatewayRequest(t *testing.T, req *http.Request, body []byte, priv ed25519.PrivateKey, keyID, nonce string, _ time.Time) {
	t.Helper()
	if body == nil {
		body = []byte{}
	}
	bodyRC := io.NopCloser(bytes.NewReader(body))
	digest, err := httpsign.GenerateContentDigestHeader(&bodyRC, []string{httpsign.DigestSha256})
	if err != nil {
		t.Fatalf("content-digest: %v", err)
	}
	req.Header.Set("Content-Digest", digest)
	req.Body = io.NopCloser(bytes.NewReader(body))

	fields := afisign.RequiredFields()
	cfg := httpsign.NewSignConfig().SetKeyID(keyID).SetNonce(nonce).SignCreated(true)
	signer, err := httpsign.NewEd25519Signer(priv, cfg, fields)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	sigInput, sig, err := httpsign.SignRequest(afisign.SignatureName, *signer, req)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req.Header.Set("Signature-Input", sigInput)
	req.Header.Set("Signature", sig)
	req.Body = io.NopCloser(bytes.NewReader(body))
}

func TestAuthenticateKey(t *testing.T) {
	raw := "sk-good"
	snap := snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			KeyHash: snapshot.HashKey(raw), ProjectID: "p1", OrganizationID: "o1",
		}},
	})
	if _, err := AuthenticateKey(snap, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateKey(snap, "sk-bad"); err != kernel.ErrUnauthorized {
		t.Fatalf("want unauthorized, got %v", err)
	}
	if _, err := AuthenticateKey(nil, raw); err != kernel.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestChatCompletionsUnauthorized(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: "http://example.invalid", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))

	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-bad")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestChatCompletionsSignedRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-test",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "pong"}},
			},
			"usage": map[string]int{"prompt_tokens": 2, "completion_tokens": 1},
		})
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	priv, pubPEM := testEd25519PublicKeyPEM(t)
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		SigningKeys: []snapshot.SigningKey{{
			ID: "sig1", KeyID: "kid-1", ProjectID: "p1", OrganizationID: "o1", Name: "svc",
			Algorithm: "ed25519", PublicKeyPEM: pubPEM, Status: "active",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: upstream.URL, APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
	var got UsageEvent
	p.Usage = func(e UsageEvent) { got = e }

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	signGatewayRequest(t, req, []byte(body), priv, "kid-1", "nonce-1", time.Now())
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.AuthMethod != snapshot.AuthMethodSignedRequest || got.SigningKeyID != "sig1" || got.SignerKeyID != "kid-1" {
		t.Fatalf("usage event: %+v", got)
	}
}

func TestSignedRequestReplayRejected(t *testing.T) {
	priv, pubPEM := testEd25519PublicKeyPEM(t)
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		SigningKeys: []snapshot.SigningKey{{
			ID: "sig1", KeyID: "kid-1", OrganizationID: "o1", Name: "svc",
			Algorithm: "ed25519", PublicKeyPEM: pubPEM, Status: "active",
		}},
	}))
	p := NewPipeline(holder, NewRegistry(), slog.Default())
	nonce := "replay-nonce"
	makeReq := func() *http.Request {
		body := []byte("{}")
		req := httptest.NewRequest(http.MethodGet, "/v1/models", bytes.NewReader(body))
		signGatewayRequest(t, req, nil, priv, "kid-1", nonce, time.Now())
		return req
	}
	rr1 := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr1, makeReq())
	if rr1.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", rr1.Code, rr1.Body.String())
	}
	rr2 := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr2, makeReq())
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("second status=%d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestChatCompletionsNonStreamMockUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-test",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "pong"}},
			},
			"usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1},
		})
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: upstream.URL, APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
	var got UsageEvent
	p.Usage = func(e UsageEvent) { got = e }

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	b, _ := io.ReadAll(rr.Body)
	if !bytes.Contains(b, []byte("pong")) {
		t.Fatalf("unexpected body: %s", b)
	}
	if got.PromptTokens != 3 || got.CompletionTokens != 1 || got.Status != "ok" {
		t.Fatalf("usage event: %+v", got)
	}
}

func TestChatCompletionsStreamParsesUsage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		opts, _ := req["stream_options"].(map[string]any)
		if opts == nil || opts["include_usage"] != true {
			http.Error(w, "missing stream_options.include_usage", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":9,\"completion_tokens\":2}}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: upstream.URL, APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
	var got UsageEvent
	p.Usage = func(e UsageEvent) { got = e }

	body := `{"model":"gpt-4o-mini","stream":true,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.PromptTokens != 9 || got.CompletionTokens != 2 {
		t.Fatalf("usage event: %+v", got)
	}
}

type stubCredOpener struct {
	secret string
}

func (s stubCredOpener) Open(context.Context, snapshot.Credential) (string, error) {
	return s.secret, nil
}

func TestChatCompletionsPolicyCredentialMarksBYOK(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-partner" {
			http.Error(w, "want partner key, got "+got, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "sk-platform")

	credCfg, _ := json.Marshal(map[string]string{
		"credential_name_expr": `request.headers["x-tenant-id"]`,
	})
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: upstream.URL, APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
		Credentials: []snapshot.Credential{{
			ID: "cred_partner", OrganizationID: "o1", Name: "acme",
			ProviderType: "openai", StorageKind: snapshot.CredentialStorageEnv,
			SecretRef: "PARTNER_KEY", Status: snapshot.CredentialStatusActive,
		}},
		Policies: []snapshot.Policy{{
			ID: "pol1", OrganizationID: "o1", Name: "byok-header",
			Expression: `("x-tenant-id" in request.headers) && request.headers["x-tenant-id"] != ""`,
			Actions: []snapshot.PolicyAction{{
				Type: snapshot.PolicyActionUseCredential, Config: credCfg,
			}},
			Enabled: true, Priority: 100,
		}},
	}))

	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
	p.Policies = ev
	p.Credentials = stubCredOpener{secret: "sk-partner"}
	var got UsageEvent
	p.Usage = func(e UsageEvent) { got = e }

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	req.Header.Set("X-Tenant-Id", "acme")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.CredentialID != "cred_partner" {
		t.Fatalf("credential_id=%q want cred_partner; event=%+v", got.CredentialID, got)
	}
	if !got.UsedBYOK {
		t.Fatalf("expected used_byok=true; event=%+v", got)
	}
}
