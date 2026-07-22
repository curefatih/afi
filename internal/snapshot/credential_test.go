package snapshot

import "testing"

func TestResolveCredentialPrefersAPIKey(t *testing.T) {
	t.Parallel()
	snap := &Snapshot{
		Credentials: map[string]Credential{
			"c_org": {ID: "c_org", ProviderType: "openai", Status: CredentialStatusActive},
			"c_key": {ID: "c_key", ProviderType: "openai", Status: CredentialStatusActive},
		},
		Assignments: map[string]string{
			AssignmentKey("openai", ScopeOrganization, "org1"): "c_org",
			AssignmentKey("openai", ScopeAPIKey, "key1"):       "c_key",
		},
	}
	c, ok := snap.ResolveCredential("openai", APIKey{ID: "key1", OrganizationID: "org1", ProjectID: "proj1"})
	if !ok || c.ID != "c_key" {
		t.Fatalf("got %+v ok=%v", c, ok)
	}
}

func TestResolveCredentialPrefersProject(t *testing.T) {
	t.Parallel()
	snap := &Snapshot{
		Credentials: map[string]Credential{
			"c_org":  {ID: "c_org", ProviderType: "openai", Status: CredentialStatusActive},
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

func TestResolveCredentialByPolicyOverride(t *testing.T) {
	t.Parallel()
	snap := &Snapshot{
		Credentials: map[string]Credential{
			"c_company": {
				ID: "c_company", OrganizationID: "org1", Name: "partner-acme",
				ProviderType: "openai", Status: CredentialStatusActive,
			},
			"c_org": {
				ID: "c_org", OrganizationID: "org1", Name: "default",
				ProviderType: "openai", Status: CredentialStatusActive,
			},
		},
		Assignments: map[string]string{
			AssignmentKey("openai", ScopeOrganization, "org1"): "c_org",
		},
	}
	key := APIKey{OrganizationID: "org1", ProjectID: "projA"}

	c, ok, missing := snap.ResolveCredentialForCall("openai", key, "")
	if missing || !ok || c.ID != "c_org" {
		t.Fatalf("no override: got %+v ok=%v missing=%v", c, ok, missing)
	}

	c, ok, missing = snap.ResolveCredentialForCall("openai", key, "partner-acme")
	if missing || !ok || c.ID != "c_company" {
		t.Fatalf("with override: got %+v ok=%v missing=%v", c, ok, missing)
	}

	_, ok, missing = snap.ResolveCredentialForCall("openai", key, "unknown")
	if ok || !missing {
		t.Fatalf("unknown: ok=%v missing=%v", ok, missing)
	}
}
