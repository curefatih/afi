package dataplane

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/adapters/objectstore"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestImagesPassThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte(`"model":"dall-e-3"`)) {
			t.Fatalf("body=%s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":1,"data":[{"b64_json":"AAAA"}]}`))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("OPENAI_API_KEY", "sk-test")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()

	raw := "sk-image-test"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-image",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_openai", Type: "openai", BaseURL: upstream.URL + "/v1",
			APIKeyEnv: "OPENAI_API_KEY", Name: "OpenAI",
			Capabilities: snapshot.DefaultCapabilities("openai"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "dall-e-3", ProviderID: "prov_openai", TargetModel: "dall-e-3",
		}},
	}))

	p := NewPipelineWithRegistry(holder, RegistryWithOpenAI(client), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(
		`{"model":"dall-e-3","prompt":"a cat"}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"b64_json"`)) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestImagesRequiresCapability(t *testing.T) {
	raw := "sk-image-cap"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-image",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_anthropic", Type: "anthropic", BaseURL: "https://api.anthropic.com",
			APIKeyEnv: "ANTHROPIC_API_KEY", Name: "Anthropic",
			Capabilities: snapshot.DefaultCapabilities("anthropic"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "dall-e-3", ProviderID: "prov_anthropic", TargetModel: "dall-e-3",
		}},
	}))
	p := NewPipelineWithRegistry(holder, NewRegistry(), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(
		`{"model":"dall-e-3","prompt":"x"}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

type memStore struct {
	objects map[string][]byte
}

func (m *memStore) Put(_ context.Context, key string, body io.Reader, _ int64, _ objectstore.PutOptions) error {
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.objects[key] = b
	return nil
}

func (m *memStore) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://assets.example/" + key, nil
}

func TestMaybePersistImages(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
	b64 := base64.StdEncoding.EncodeToString(png)
	body := []byte(`{"created":1,"data":[{"b64_json":"` + b64 + `"}]}`)

	store := &memStore{}
	rewritten, err := persistImagesWithStore(context.Background(), store, time.Hour, snapshot.APIKey{
		OrganizationID: "o1", ProjectID: "p1",
	}, body, false)
	if err != nil {
		t.Fatal(err)
	}
	if rewritten == nil {
		t.Fatal("expected rewrite")
	}
	if !bytes.Contains(rewritten, []byte(`https://assets.example/`)) {
		t.Fatalf("body=%s", rewritten)
	}
	if len(store.objects) != 1 {
		t.Fatalf("objects=%d", len(store.objects))
	}
	for k := range store.objects {
		if !strings.HasPrefix(k, "o1/p1/") || !strings.HasSuffix(k, ".png") {
			t.Fatalf("key=%s", k)
		}
	}
}

// persistImagesWithStore is a test seam wrapping maybePersistImages store write path.
func persistImagesWithStore(ctx context.Context, store objectstore.Store, ttl time.Duration, key snapshot.APIKey, body []byte, wantB64 bool) ([]byte, error) {
	p := &Pipeline{HTTP: http.DefaultClient}
	return p.rewritePersistedImages(ctx, store, ttl, key, body, wantB64)
}
