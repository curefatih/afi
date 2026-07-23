package platform

import "testing"

func TestActorContext(t *testing.T) {
	ctx := WithActor(t.Context(), "user_1")
	if got := ActorFrom(ctx); got != "user_1" {
		t.Fatalf("got %q", got)
	}
	if got := ActorFrom(t.Context()); got != "" {
		t.Fatalf("empty ctx: got %q", got)
	}
}
