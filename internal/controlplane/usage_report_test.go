package controlplane

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/curefatih/afi/internal/usage"
)

func TestUsageReportObserverDoesNotError(t *testing.T) {
	o := NewUsageReportObserver(slog.Default())
	payload, _ := json.Marshal(usage.Event{
		OrganizationID: "org_1", Model: "gpt-4o-mini", Status: "ok",
		PromptTokens: 10, CompletionTokens: 5, Modality: "chat",
		Tags: map[string]string{"region": "eu-west", "deployment_id": "dep_1"},
	})
	if err := o.Enqueue(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
}
