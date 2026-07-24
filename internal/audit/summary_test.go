package audit

import "testing"

func TestSummary(t *testing.T) {
	if got := Summary("quota.created", "quota_1"); got != "Created quota quota_1" {
		t.Fatalf("got %q", got)
	}
	if got := Summary("environment.created", "env_1"); got != "Created environment env_1" {
		t.Fatalf("got %q", got)
	}
	if got := Summary("environment.deleted", "env_1"); got != "Deleted environment env_1" {
		t.Fatalf("got %q", got)
	}
	if got := Summary("snapshot.published", ""); got != "Published gateway snapshot" {
		t.Fatalf("got %q", got)
	}
}
