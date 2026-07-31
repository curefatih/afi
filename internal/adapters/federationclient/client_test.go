package federationclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestClientJoinAndExport(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/v1/federation/peers/join", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(federation.ControlPlanePeer{ID: "peer_1", Status: federation.PeerStatusActive})
	})
	mux.HandleFunc("GET /internal/v1/federation/regions/{slug}/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(TokenHeader) != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(federation.RegionExport{
			Revision: 2, RegionID: "reg_1", RegionSlug: "eu-west",
			Memberships: []regions.OrgRegionMembership{},
			Overlays:    []regions.RegionConfigOverlay{},
			Snapshot:    &snapshot.Snapshot{Version: 7},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL, "tok")
	peer, err := c.Join(context.Background())
	if err != nil || peer.ID != "peer_1" {
		t.Fatalf("%+v %v", peer, err)
	}
	exp, err := c.Export(context.Background(), "eu-west", 0)
	if err != nil || exp.Revision != 2 || exp.Snapshot == nil {
		t.Fatalf("%+v %v", exp, err)
	}
}
