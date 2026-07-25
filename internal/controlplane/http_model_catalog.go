package controlplane

import (
	"net/http"

	"github.com/curefatih/afi/internal/modelcatalog"
)

type modelCatalogDTO struct {
	ProviderType string `json:"provider_type"`
	ID           string `json:"id"`
	Mode         string `json:"mode"`
}

func (s *Server) handleListModelCatalog(w http.ResponseWriter, r *http.Request) {
	providerType := r.URL.Query().Get("provider_type")
	listed := modelcatalog.List(providerType)
	out := make([]modelCatalogDTO, 0, len(listed))
	for _, m := range listed {
		out = append(out, modelCatalogDTO{
			ProviderType: m.ProviderType,
			ID:           m.ID,
			Mode:         m.Mode,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
