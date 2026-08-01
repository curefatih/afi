package hubclient

import (
	"context"
	"testing"
	"time"
)

func TestNewTrims(t *testing.T) {
	c := New(" http://cp.example/ ", " tok ")
	if c.BaseURL != "http://cp.example" || c.JoinToken != "tok" {
		t.Fatalf("%+v", c)
	}
}

func TestHeartbeatNotConfigured(t *testing.T) {
	c := New("", "")
	err := c.Heartbeat(context.Background(), "dep", 1, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunHeartbeatLoopCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunHeartbeatLoop(ctx, New("", ""), "dep", time.Hour, func() int64 { return 0 }, "", nil)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not exit")
	}
}
