package httpsign

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"

	lib "github.com/yaronf/httpsign"
)

// SignatureName is the preferred Signature / Signature-Input dictionary label.
const SignatureName = "sig1"

// RequiredComponents are the covered components the gateway requires.
var RequiredComponents = []string{"@method", "@path", "@query", "content-digest"}

// RequiredFields returns the httpsign Fields list used for signing and verifying.
func RequiredFields() lib.Fields {
	return lib.Headers(RequiredComponents...)
}

// ParsePrivateKeyPEM parses a PKCS#8 or PKIX Ed25519 private key PEM.
func ParsePrivateKeyPEM(pemBytes []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM private key")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		priv, ok := key.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PEM is not an ed25519 private key")
		}
		return priv, nil
	}
	// Some tools emit "PRIVATE KEY" that ParsePKCS8 already covers; raw seed is uncommon.
	return nil, fmt.Errorf("parse private key: unsupported PEM type %q", block.Type)
}

// SignRequest adds Content-Digest, Signature-Input, and Signature headers to req.
// The request body is read (if present), digested, restored, and signed.
// nonce may be empty to generate a random nonce.
func SignRequest(req *http.Request, privateKey ed25519.PrivateKey, keyID string, nonce string) error {
	if req == nil {
		return fmt.Errorf("nil request")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid ed25519 private key")
	}
	if keyID == "" {
		return fmt.Errorf("key id is required")
	}
	if nonce == "" {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			return fmt.Errorf("nonce: %w", err)
		}
		nonce = hex.EncodeToString(b[:])
	}

	body, err := readAndRestoreBody(req)
	if err != nil {
		return err
	}
	bodyRC := io.NopCloser(bytes.NewReader(body))
	digest, err := lib.GenerateContentDigestHeader(&bodyRC, []string{lib.DigestSha256})
	if err != nil {
		return fmt.Errorf("content-digest: %w", err)
	}
	req.Header.Set("Content-Digest", digest)
	req.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) > 0 {
		req.ContentLength = int64(len(body))
	}

	cfg := lib.NewSignConfig().SetKeyID(keyID).SetNonce(nonce).SignCreated(true)
	signer, err := lib.NewEd25519Signer(privateKey, cfg, RequiredFields())
	if err != nil {
		return err
	}
	sigInput, sig, err := lib.SignRequest(SignatureName, *signer, req)
	if err != nil {
		return err
	}
	req.Header.Set("Signature-Input", sigInput)
	req.Header.Set("Signature", sig)
	req.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

func readAndRestoreBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return []byte{}, nil
	}
	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// Transport is an http.RoundTripper that signs each request before forwarding.
type Transport struct {
	Base       http.RoundTripper
	PrivateKey ed25519.PrivateKey
	KeyID      string
}

// RoundTrip implements http.RoundTripper.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t == nil {
		return nil, fmt.Errorf("nil transport")
	}
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	// Clone so we don't mutate the caller's request headers/body unexpectedly
	// beyond what RoundTrippers normally do; we must sign a copy with a body.
	clone := req.Clone(req.Context())
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		clone.Body = io.NopCloser(bytes.NewReader(body))
		clone.ContentLength = int64(len(body))
	}
	if err := SignRequest(clone, t.PrivateKey, t.KeyID, ""); err != nil {
		return nil, err
	}
	return base.RoundTrip(clone)
}

// Client returns an *http.Client that signs outbound requests.
func Client(privateKey ed25519.PrivateKey, keyID string) *http.Client {
	return &http.Client{Transport: &Transport{PrivateKey: privateKey, KeyID: keyID}}
}
