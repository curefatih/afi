package snapshot

import "testing"

func TestCompileIncludesQuotas(t *testing.T) {
	t.Parallel()
	snap := Compile(Source{
		Quotas: []Quota{{
			ID: "q1", OrganizationID: "org_a", ScopeType: "project", ScopeID: "proj_1",
			Metric: "requests", LimitValue: 10, Window: "total",
		}},
	})
	if len(snap.Quotas) != 1 || snap.Quotas[0].LimitValue != 10 {
		t.Fatalf("quotas: %+v", snap.Quotas)
	}
}

func TestResolveQuotaPrefersMostSpecific(t *testing.T) {
	t.Parallel()
	snap := Compile(Source{
		Quotas: []Quota{
			{ID: "org", OrganizationID: "o1", ScopeType: ScopeOrganization, ScopeID: "o1", Metric: MetricRequests, LimitValue: 100, Window: WindowTotal},
			{ID: "proj", OrganizationID: "o1", ScopeType: ScopeProject, ScopeID: "p1", Metric: MetricRequests, LimitValue: 50, Window: WindowTotal},
			{ID: "key", OrganizationID: "o1", ScopeType: ScopeAPIKey, ScopeID: "k1", Metric: MetricRequests, LimitValue: 5, Window: WindowTotal},
		},
	})
	key := APIKey{ID: "k1", ProjectID: "p1", OrganizationID: "o1"}
	q, ok := snap.ResolveQuota(key, MetricRequests)
	if !ok || q.ID != "key" || q.LimitValue != 5 {
		t.Fatalf("want key quota, got %+v ok=%v", q, ok)
	}
}

func TestResolveQuotaFallsBackToProject(t *testing.T) {
	t.Parallel()
	snap := Compile(Source{
		Quotas: []Quota{
			{ID: "org", OrganizationID: "o1", ScopeType: ScopeOrganization, ScopeID: "o1", Metric: MetricRequests, LimitValue: 100, Window: WindowTotal},
			{ID: "proj", OrganizationID: "o1", ScopeType: ScopeProject, ScopeID: "p1", Metric: MetricRequests, LimitValue: 50, Window: WindowTotal},
		},
	})
	key := APIKey{ID: "k1", ProjectID: "p1", OrganizationID: "o1"}
	q, ok := snap.ResolveQuota(key, MetricRequests)
	if !ok || q.ID != "proj" {
		t.Fatalf("want project quota, got %+v ok=%v", q, ok)
	}
}

func TestResolveQuotaNone(t *testing.T) {
	t.Parallel()
	snap := Compile(Source{})
	_, ok := snap.ResolveQuota(APIKey{ID: "k", ProjectID: "p", OrganizationID: "o"}, MetricRequests)
	if ok {
		t.Fatal("expected no quota")
	}
}

func TestResolveQuotaPrefersUserOverOrgForPersonalKey(t *testing.T) {
	t.Parallel()
	snap := Compile(Source{
		Quotas: []Quota{
			{ID: "org", OrganizationID: "o1", ScopeType: ScopeOrganization, ScopeID: "o1", Metric: MetricRequests, LimitValue: 100, Window: WindowTotal},
			{ID: "user", OrganizationID: "o1", ScopeType: ScopeUser, ScopeID: "u1", Metric: MetricRequests, LimitValue: 20, Window: WindowTotal},
		},
	})
	key := APIKey{ID: "k1", OrganizationID: "o1", Kind: KeyKindPersonal, OwnerUserID: "u1"}
	q, ok := snap.ResolveQuota(key, MetricRequests)
	if !ok || q.ID != "user" {
		t.Fatalf("want user quota, got %+v ok=%v", q, ok)
	}
}

func TestResolveQuotaUserSkippedForServiceAccount(t *testing.T) {
	t.Parallel()
	snap := Compile(Source{
		Quotas: []Quota{
			{ID: "org", OrganizationID: "o1", ScopeType: ScopeOrganization, ScopeID: "o1", Metric: MetricRequests, LimitValue: 100, Window: WindowTotal},
			{ID: "user", OrganizationID: "o1", ScopeType: ScopeUser, ScopeID: "u1", Metric: MetricRequests, LimitValue: 20, Window: WindowTotal},
		},
	})
	key := APIKey{ID: "k1", OrganizationID: "o1", ProjectID: "p1", Kind: KeyKindServiceAccount}
	q, ok := snap.ResolveQuota(key, MetricRequests)
	if !ok || q.ID != "org" {
		t.Fatalf("want org quota for SA, got %+v ok=%v", q, ok)
	}
}

func TestResolveQuotaAPIKeyBeatsUser(t *testing.T) {
	t.Parallel()
	snap := Compile(Source{
		Quotas: []Quota{
			{ID: "user", OrganizationID: "o1", ScopeType: ScopeUser, ScopeID: "u1", Metric: MetricRequests, LimitValue: 20, Window: WindowTotal},
			{ID: "key", OrganizationID: "o1", ScopeType: ScopeAPIKey, ScopeID: "k1", Metric: MetricRequests, LimitValue: 3, Window: WindowTotal},
		},
	})
	key := APIKey{ID: "k1", OrganizationID: "o1", Kind: KeyKindPersonal, OwnerUserID: "u1"}
	q, ok := snap.ResolveQuota(key, MetricRequests)
	if !ok || q.ID != "key" {
		t.Fatalf("want api_key quota, got %+v ok=%v", q, ok)
	}
}
