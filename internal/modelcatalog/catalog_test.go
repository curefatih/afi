package modelcatalog

import (
	"testing"

	"github.com/curefatih/afi/internal/usage"
)

func TestLookupChatAndAudio(t *testing.T) {
	t.Parallel()
	chat, ok := Lookup("openai", "gpt-4o-mini")
	if !ok || !chat.IsChat() || chat.MaxInputTokens != 128000 {
		t.Fatalf("chat=%+v ok=%v", chat, ok)
	}
	if !chat.StreamingEnabled() {
		t.Fatal("expected streaming")
	}
	tts, ok := Lookup("openai", "tts-1")
	if !ok || !tts.IsTTS() || tts.StreamingEnabled() {
		t.Fatalf("tts=%+v ok=%v", tts, ok)
	}
	stt, ok := Lookup("openai_compatible", "whisper-1")
	if !ok || !stt.IsSTT() {
		t.Fatalf("stt alias=%+v ok=%v", stt, ok)
	}
	if _, ok := Lookup("openai", "does-not-exist"); ok {
		t.Fatal("expected miss")
	}
}

func TestEstimateCostChat(t *testing.T) {
	t.Parallel()
	got := EstimateCostUSD(usage.Event{
		ProviderType: "openai", TargetModel: "gpt-4o-mini",
		PromptTokens: 1_000_000, CompletionTokens: 0,
	})
	if got == nil || *got < 0.149 || *got > 0.151 {
		t.Fatalf("got %v", got)
	}
}

func TestEstimateCostTTS(t *testing.T) {
	t.Parallel()
	got := EstimateCostUSD(usage.Event{
		ProviderType: "openai", TargetModel: "tts-1",
		Modality: "tts",
		Metrics:  map[string]any{"characters": 1000},
	})
	// 1000 * 0.000015 = 0.015
	if got == nil || *got < 0.0149 || *got > 0.0151 {
		t.Fatalf("got %v", got)
	}
}

func TestEstimateCostSTT(t *testing.T) {
	t.Parallel()
	got := EstimateCostUSD(usage.Event{
		ProviderType: "openai", TargetModel: "whisper-1",
		Modality: "stt",
		Metrics:  map[string]any{"audio_seconds": 60.0},
	})
	// 60 * 0.0001 = 0.006
	if got == nil || *got < 0.0059 || *got > 0.0061 {
		t.Fatalf("got %v", got)
	}
}

func TestEstimateCostMissingMetrics(t *testing.T) {
	t.Parallel()
	if got := EstimateCostUSD(usage.Event{
		ProviderType: "openai", TargetModel: "tts-1",
	}); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
