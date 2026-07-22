package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Hashicorp resolves hashicorp://path[#jsonKey] via the Vault HTTP API (KV v1/v2 compatible).
// Auth: VAULT_TOKEN (or AFI_SECRETS_VAULT_TOKEN). Address: VAULT_ADDR / AFI_SECRETS_VAULT_ADDR.
type Hashicorp struct {
	Addr   string
	Token  string
	Client *http.Client
}

// NewHashicorpFromEnv builds a client when address is configured; returns nil,nil when disabled.
func NewHashicorpFromEnv(addr, token string) *Hashicorp {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv("VAULT_ADDR"))
	}
	if addr == "" {
		return nil
	}
	token = strings.TrimSpace(token)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("VAULT_TOKEN"))
	}
	if token == "" {
		token = strings.TrimSpace(os.Getenv("AFI_SECRETS_VAULT_TOKEN"))
	}
	return &Hashicorp{
		Addr:  strings.TrimRight(addr, "/"),
		Token: token,
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (h *Hashicorp) Get(ctx context.Context, ref string) (string, error) {
	if h == nil || h.Addr == "" {
		return "", fmt.Errorf("hashicorp vault not configured")
	}
	if h.Token == "" {
		return "", fmt.Errorf("hashicorp vault token not configured")
	}
	p, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	if p.Scheme != "hashicorp" {
		return "", fmt.Errorf("expected hashicorp:// ref, got %q", ref)
	}
	path := strings.TrimPrefix(p.Path, "/")
	// Prefer KV v2 read shape: /v1/<mount>/data/<path>
	apiURL := h.Addr + "/v1/" + path
	if !strings.Contains(path, "/data/") {
		// If caller passed secret/afi/openai, try secret/data/afi/openai
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			apiURL = h.Addr + "/v1/" + parts[0] + "/data/" + parts[1]
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", h.Token)
	resp, err := h.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("vault %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", fmt.Errorf("vault decode: %w", err)
	}
	// KV v2: { data: { data: { key: val }, metadata: ... } }
	// KV v1: { data: { key: val } }
	var inner map[string]any
	if err := json.Unmarshal(envelope.Data, &inner); err != nil {
		return "", fmt.Errorf("vault data decode: %w", err)
	}
	data := inner
	if nested, ok := inner["data"].(map[string]any); ok {
		data = nested
	}
	if p.Key != "" {
		v, ok := data[p.Key]
		if !ok {
			return "", fmt.Errorf("vault secret missing key %q", p.Key)
		}
		return stringifySecret(v)
	}
	if v, ok := data["value"]; ok {
		return stringifySecret(v)
	}
	if len(data) == 1 {
		for _, v := range data {
			return stringifySecret(v)
		}
	}
	return "", fmt.Errorf("vault secret has multiple keys; specify #jsonKey in ref")
}

func stringifySecret(v any) (string, error) {
	switch t := v.(type) {
	case string:
		if t == "" {
			return "", fmt.Errorf("empty secret value")
		}
		return t, nil
	case float64, bool:
		return fmt.Sprint(t), nil
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
