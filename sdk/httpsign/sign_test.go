package httpsign_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/sdk/httpsign"
	lib "github.com/yaronf/httpsign"
)

func testKey(t *testing.T) (ed25519.PrivateKey, ed25519.PublicKey, []byte) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return priv, pub, pemBytes
}

func TestSignRequestRoundTripVerify(t *testing.T) {
	priv, pub, pemBytes := testKey(t)
	parsed, err := httpsign.ParsePrivateKeyPEM(pemBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed, priv) {
		t.Fatal("parsed key mismatch")
	}

	body := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "http://gateway.example/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if err := httpsign.SignRequest(req, priv, "svc-1", "nonce-abc"); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Signature") == "" || req.Header.Get("Signature-Input") == "" {
		t.Fatalf("missing signature headers: %v", req.Header)
	}
	if req.Header.Get("Content-Digest") == "" {
		t.Fatal("missing Content-Digest")
	}
	gotBody, _ := io.ReadAll(req.Body)
	if !bytes.Equal(gotBody, body) {
		t.Fatalf("body not restored: %q", gotBody)
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	cfg := lib.NewVerifyConfig().
		SetKeyID("svc-1").
		SetAllowedAlgs([]string{"ed25519"}).
		SetVerifyCreated(true).
		SetNonceValidator(func(string) error { return nil })
	verifier, err := lib.NewEd25519Verifier(pub, cfg, httpsign.RequiredFields())
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.VerifyRequest(httpsign.SignatureName, *verifier, req); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestClientTransportSigns(t *testing.T) {
	priv, pub, _ := testKey(t)
	var sawSig bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawSig = r.Header.Get("Signature") != ""
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		cfg := lib.NewVerifyConfig().
			SetKeyID("kid").
			SetAllowedAlgs([]string{"ed25519"}).
			SetVerifyCreated(true).
			SetNonceValidator(func(string) error { return nil })
		verifier, err := lib.NewEd25519Verifier(pub, cfg, httpsign.RequiredFields())
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err := lib.VerifyRequest(httpsign.SignatureName, *verifier, r); err != nil {
			http.Error(w, err.Error(), 401)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	client := httpsign.Client(priv, "kid")
	res, err := client.Post(upstream.URL+"/v1/models", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s sawSig=%v", res.StatusCode, b, sawSig)
	}
	if !sawSig {
		t.Fatal("upstream did not see Signature header")
	}
}
