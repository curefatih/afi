package postgres

import (
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestDecodeCapabilities(t *testing.T) {
	t.Parallel()
	caps := DecodeCapabilities("openai", []byte(`{"chat":true,"stream":true}`))
	if !caps.Chat || !caps.Stream || !caps.TTS || !caps.STT {
		t.Fatalf("caps=%+v", caps)
	}
	empty := DecodeCapabilities("openai", nil)
	def := snapshot.DefaultCapabilities("openai")
	if empty != def {
		t.Fatalf("empty caps=%+v want %+v", empty, def)
	}
}

func TestDecodeFallbacks(t *testing.T) {
	t.Parallel()
	fb := DecodeFallbacks([]byte(`[{"provider_id":"p2","target_model":"m2"}]`))
	if len(fb) != 1 || fb[0].ProviderID != "p2" || fb[0].TargetModel != "m2" {
		t.Fatalf("fb=%+v", fb)
	}
	weighted := DecodeFallbacks([]byte(`[{"provider_id":"p2","target_model":"m2","weight":3}]`))
	if len(weighted) != 1 || weighted[0].Weight != 3 {
		t.Fatalf("weighted=%+v", weighted)
	}
	if got := DecodeFallbacks(nil); len(got) != 0 {
		t.Fatalf("nil fallbacks=%+v", got)
	}
	if got := DecodeFallbacks([]byte(`not-json`)); len(got) != 0 {
		t.Fatalf("bad json fallbacks=%+v", got)
	}
}

func TestDecodeRetry(t *testing.T) {
	t.Parallel()
	got := DecodeRetry([]byte(`{"max_attempts":3,"backoff":{"strategy":"fixed","base_delay":"100ms"}}`))
	if got == nil || got.MaxAttempts != 3 || got.Backoff.Strategy != "fixed" || got.Backoff.BaseDelay != "100ms" {
		t.Fatalf("got=%+v", got)
	}
	if DecodeRetry(nil) != nil {
		t.Fatal("nil raw should decode to nil")
	}
	if DecodeRetry([]byte(`null`)) != nil {
		t.Fatal("null json should decode to nil")
	}
	if DecodeRetry([]byte(`not-json`)) != nil {
		t.Fatal("bad json should decode to nil")
	}
}
