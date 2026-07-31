package controlplane

import (
	"testing"
	"time"

	"github.com/curefatih/afi/internal/usage"
)

func TestMergeUsageEventsOrdersAndLimits(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	local := []UsageEvent{
		{ID: 1, OrganizationID: "org", CreatedAt: now.Add(-2 * time.Hour), Model: "local"},
	}
	remote := []UsageEvent{
		{ID: 2, OrganizationID: "org", CreatedAt: now.Add(-1 * time.Hour), Model: "remote", Tags: map[string]string{"region": "eu-west"}},
		{ID: 3, OrganizationID: "org", CreatedAt: now, Model: "remote-new", Tags: map[string]string{"region": "eu-west"}},
	}
	got := mergeUsageEvents(local, remote, 2)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Model != "remote-new" || got[1].Model != "remote" {
		t.Fatalf("order=%v %v", got[0].Model, got[1].Model)
	}
}

func TestUsageMatchesFilter(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	e := usage.Record{
		OrganizationID: "org_1",
		Model:          "gpt",
		Modality:       "chat",
		CreatedAt:      time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
	}
	if !usageMatchesFilter(e, "org_1", UsageFilter{From: &from, To: &to, Model: "gpt"}) {
		t.Fatal("expected match")
	}
	if usageMatchesFilter(e, "org_2", UsageFilter{}) {
		t.Fatal("org mismatch")
	}
	if usageMatchesFilter(e, "org_1", UsageFilter{Model: "other"}) {
		t.Fatal("model mismatch")
	}
}

func TestSummarizeAndMergeUsageSummary(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	cost := 0.5
	remote := []UsageEvent{
		{ID: 1, Model: "m1", Modality: "chat", CreatedAt: day, PromptTokens: 10, CompletionTokens: 5, CostUSD: &cost},
		{ID: 2, Model: "m1", Modality: "chat", CreatedAt: day, PromptTokens: 3, CompletionTokens: 1},
	}
	sum := summarizeUsageRecords(remote, "day")
	if len(sum) != 1 || sum[0].Requests != 2 || sum[0].PromptTokens != 13 {
		t.Fatalf("sum=%+v", sum)
	}
	local := []UsageSummaryBucket{{
		Bucket: "2026-07-15", Label: "2026-07-15", Requests: 4, PromptTokens: 40,
	}}
	merged := mergeUsageSummary(local, sum)
	if len(merged) != 1 || merged[0].Requests != 6 || merged[0].PromptTokens != 53 {
		t.Fatalf("merged=%+v", merged)
	}
}
