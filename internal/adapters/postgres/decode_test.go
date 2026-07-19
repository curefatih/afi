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
	if got := DecodeFallbacks(nil); len(got) != 0 {
		t.Fatalf("nil fallbacks=%+v", got)
	}
	if got := DecodeFallbacks([]byte(`not-json`)); len(got) != 0 {
		t.Fatalf("bad json fallbacks=%+v", got)
	}
}
