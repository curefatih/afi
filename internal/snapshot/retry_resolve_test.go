package snapshot

import "testing"

func TestResolveRetry(t *testing.T) {
	t.Parallel()
	routeRetry := &RetryConfig{MaxAttempts: 5, Backoff: BackoffConfig{Strategy: BackoffFixed, BaseDelay: "10ms"}}
	orgRetry := &RetryConfig{MaxAttempts: 2, Backoff: BackoffConfig{Strategy: BackoffFixed, BaseDelay: "1ms"}}
	s := Compile(Source{
		DefaultRetries: map[string]*RetryConfig{"org1": orgRetry},
	})

	got := s.ResolveRetry(Route{OrganizationID: "org1", Retry: routeRetry})
	if got == nil || got.MaxAttempts != 5 {
		t.Fatalf("route override=%+v", got)
	}
	got = s.ResolveRetry(Route{OrganizationID: "org1"})
	if got == nil || got.MaxAttempts != 2 {
		t.Fatalf("org default=%+v", got)
	}
	got = s.ResolveRetry(Route{OrganizationID: "other"})
	if got != nil {
		t.Fatalf("no default=%+v", got)
	}
}
