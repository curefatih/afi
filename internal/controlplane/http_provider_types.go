package controlplane

import (
	"net/http"

	"github.com/curefatih/afi/internal/providercatalog"
)

type providerTypeDTO struct {
	Type           string                         `json:"type"`
	Name           string                         `json:"name"`
	BaseURL        string                         `json:"base_url"`
	APIKeyEnv      string                         `json:"api_key_env"`
	AuthMode       string                         `json:"auth_mode"`
	Capabilities   providercatalog.Capabilities   `json:"capabilities"`
}

func (s *Server) handleListProviderTypes(w http.ResponseWriter, r *http.Request) {
	specs := providercatalog.UIVisible()
	out := make([]providerTypeDTO, 0, len(specs))
	for _, spec := range specs {
		out = append(out, providerTypeDTO{
			Type:         spec.Type,
			Name:         spec.DisplayName,
			BaseURL:      spec.DefaultBaseURL,
			APIKeyEnv:    spec.DefaultAPIKeyEnv,
			AuthMode:     string(spec.AuthMode),
			Capabilities: spec.Capabilities,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
