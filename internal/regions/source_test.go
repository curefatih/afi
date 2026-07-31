package regions

import (
	"context"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestBuildRegionSourceUnboundOmitted(t *testing.T) {
	base := snapshot.Source{
		APIKeys: []snapshot.APIKey{
			{ID: "k1", KeyHash: "h1", OrganizationID: "org_a", Name: "a"},
			{ID: "k2", KeyHash: "h2", OrganizationID: "org_b", Name: "b"},
		},
		Routes: []snapshot.Route{
			{OrganizationID: "org_a", Model: "m", ProviderID: "p1", TargetModel: "m"},
			{OrganizationID: "org_b", Model: "m", ProviderID: "p2", TargetModel: "m"},
		},
		Providers: []snapshot.Provider{
			{ID: "p1", Type: "openai", Name: "oai"},
			{ID: "p2", Type: "openai", Name: "oai2"},
		},
	}
	src, allowed := BuildRegionSource(base, []OrgRegionMembership{
		{OrganizationID: "org_a", RegionID: "r1", Status: MembershipStatusActive},
	}, nil)
	if len(allowed) != 1 || allowed[0] != "org_a" {
		t.Fatalf("allowed=%v", allowed)
	}
	if len(src.APIKeys) != 1 || src.APIKeys[0].OrganizationID != "org_a" {
		t.Fatalf("keys=%+v", src.APIKeys)
	}
	if len(src.Routes) != 1 || src.Routes[0].OrganizationID != "org_a" {
		t.Fatalf("routes=%+v", src.Routes)
	}
	if len(src.Providers) != 1 || src.Providers[0].ID != "p1" {
		t.Fatalf("providers=%+v", src.Providers)
	}
}

func TestBuildRegionSourceOverlayReplace(t *testing.T) {
	base := snapshot.Source{
		APIKeys: []snapshot.APIKey{
			{ID: "k1", KeyHash: "h1", OrganizationID: "org_a", Name: "a"},
		},
		Routes: []snapshot.Route{
			{OrganizationID: "org_a", Model: "base-model", ProviderID: "p1", TargetModel: "base-model"},
		},
		Providers: []snapshot.Provider{
			{ID: "p1", Type: "openai", Name: "base"},
			{ID: "p_ov", Type: "openai", Name: "overlay"},
		},
	}
	ov := RegionConfigOverlay{
		OrganizationID: "org_a",
		RegionID:       "r1",
		Payload: OverlayPayload{
			Routes: []snapshot.Route{
				{OrganizationID: "org_a", Model: "eu-model", ProviderID: "p_ov", TargetModel: "eu-model"},
			},
			Providers: []snapshot.Provider{
				{ID: "p_ov", Type: "openai", Name: "overlay"},
			},
		},
	}
	src, _ := BuildRegionSource(base, []OrgRegionMembership{
		{OrganizationID: "org_a", RegionID: "r1", Status: MembershipStatusActive},
	}, []RegionConfigOverlay{ov})
	if len(src.Routes) != 1 || src.Routes[0].Model != "eu-model" {
		t.Fatalf("routes=%+v", src.Routes)
	}
	if len(src.APIKeys) != 1 {
		t.Fatalf("keys should inherit from base, got %+v", src.APIKeys)
	}
	if len(src.Providers) != 1 || src.Providers[0].ID != "p_ov" {
		t.Fatalf("providers=%+v", src.Providers)
	}
}

func TestBuildRegionSourceDisabledMembership(t *testing.T) {
	base := snapshot.Source{
		APIKeys: []snapshot.APIKey{{ID: "k1", KeyHash: "h1", OrganizationID: "org_a"}},
	}
	src, allowed := BuildRegionSource(base, []OrgRegionMembership{
		{OrganizationID: "org_a", RegionID: "r1", Status: MembershipStatusDisabled},
	}, nil)
	if len(allowed) != 0 {
		t.Fatalf("allowed=%v", allowed)
	}
	if len(src.APIKeys) != 0 {
		t.Fatalf("keys=%v", src.APIKeys)
	}
}

func TestBindRequiresActiveRegion(t *testing.T) {
	repo := newMemRepo()
	r, err := CreateRegion(context.Background(), repo, "reg_1", "eu-west", "EU")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateRegion(context.Background(), repo, r.ID, "", RegionStatusDisabled); err != nil {
		t.Fatal(err)
	}
	_, err = BindOrgToRegion(context.Background(), repo, r.ID, "org_a", MembershipStatusActive)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPutOverlayRequiresMembership(t *testing.T) {
	repo := newMemRepo()
	if _, err := CreateRegion(context.Background(), repo, "reg_1", "us-east", "US"); err != nil {
		t.Fatal(err)
	}
	_, err := PutOverlay(context.Background(), repo, "reg_1", "org_a", OverlayPayload{})
	if err == nil {
		t.Fatal("expected error")
	}
}
