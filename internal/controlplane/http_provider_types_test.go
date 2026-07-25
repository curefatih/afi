package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListProviderTypes(t *testing.T) {
	t.Parallel()
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/provider-types", nil)
	rr := httptest.NewRecorder()
	s.handleListProviderTypes(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list []providerTypeDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	byType := map[string]providerTypeDTO{}
	for _, item := range list {
		byType[item.Type] = item
	}
	for _, typ := range []string{"openai", "anthropic", "gemini", "bedrock", "elevenlabs", "openai_compatible", "echo"} {
		if _, ok := byType[typ]; !ok {
			t.Fatalf("missing type %q in %v", typ, byType)
		}
	}
	if byType["bedrock"].AuthMode != "optional" {
		t.Fatalf("bedrock auth_mode=%q", byType["bedrock"].AuthMode)
	}
	if byType["bedrock"].Name == "" || byType["elevenlabs"].APIKeyEnv != "ELEVENLABS_API_KEY" {
		t.Fatalf("incomplete catalog entries: %+v", byType)
	}
}
