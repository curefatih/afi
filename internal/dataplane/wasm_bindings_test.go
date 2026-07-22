package dataplane

import (
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestFilterWasmHooks(t *testing.T) {
	snap := &snapshot.Snapshot{WasmHooks: []snapshot.WasmHook{
		{ID: "1", OrganizationID: "o1", Name: "b", Phase: snapshot.WasmPhaseBeforeCall, Enabled: true, Priority: 10},
		{ID: "2", OrganizationID: "o1", Name: "a", Phase: snapshot.WasmPhaseBeforeCall, Enabled: true, Priority: 20},
		{ID: "3", OrganizationID: "o1", Name: "x", Phase: snapshot.WasmPhaseBeforeCall, Enabled: false, Priority: 99},
		{ID: "4", OrganizationID: "o2", Name: "z", Phase: snapshot.WasmPhaseBeforeCall, Enabled: true, Priority: 50},
		{ID: "5", OrganizationID: "o1", Name: "c", Phase: snapshot.WasmPhaseBeforeChat, Enabled: true, Priority: 1},
	}}
	got := filterWasmHooks(snap, "o1", snapshot.WasmPhaseBeforeCall)
	if len(got) != 2 || got[0].ID != "2" || got[1].ID != "1" {
		t.Fatalf("%+v", got)
	}
}
