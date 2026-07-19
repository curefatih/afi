package snapshot

import "testing"

func TestResolveCredentialPrefersProject(t *testing.T) {
	t.Parallel()
	snap := &Snapshot{
		Credentials: map[string]Credential{
			"c_org": {ID: "c_org", ProviderType: "openai", Status: CredentialStatusActive},
			"c_proj": {ID: "c_proj", ProviderType: "openai", Status: CredentialStatusActive},
		},
		Assignments: map[string]string{
			AssignmentKey("openai", ScopeOrganization, "org1"): "c_org",
			AssignmentKey("openai", ScopeProject, "proj1"):     "c_proj",
		},
	}
	c, ok := snap.ResolveCredential("openai", APIKey{OrganizationID: "org1", ProjectID: "proj1"})
	if !ok || c.ID != "c_proj" {
		t.Fatalf("got %+v ok=%v", c, ok)
	}
}

func TestResolveCredentialFallsBackToOrg(t *testing.T) {
	t.Parallel()
	snap := &Snapshot{
		Credentials: map[string]Credential{
			"c_org": {ID: "c_org", ProviderType: "openai", Status: CredentialStatusActive},
		},
		Assignments: map[string]string{
			AssignmentKey("openai", ScopeOrganization, "org1"): "c_org",
		},
	}
	c, ok := snap.ResolveCredential("openai", APIKey{OrganizationID: "org1", ProjectID: "proj1"})
	if !ok || c.ID != "c_org" {
		t.Fatalf("got %+v ok=%v", c, ok)
	}
}
