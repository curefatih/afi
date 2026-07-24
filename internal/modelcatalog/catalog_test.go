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
	emb, ok := Lookup("openai", "text-embedding-3-small")
	if !ok || !emb.IsEmbedding() || emb.InputCostPerMTok == nil {
		t.Fatalf("emb=%+v ok=%v", emb, ok)
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

func TestEstimateCostEmbedding(t *testing.T) {
	t.Parallel()
	got := EstimateCostUSD(usage.Event{
		ProviderType: "openai", TargetModel: "text-embedding-3-small",
		Modality: "embedding", PromptTokens: 1_000_000,
	})
	// 1MTok * 0.02 = 0.02
	if got == nil || *got < 0.019 || *got > 0.021 {
		t.Fatalf("got %v", got)
	}
}

func TestEstimateCostImage(t *testing.T) {
	t.Parallel()
	img, ok := Lookup("openai", "dall-e-3")
	if !ok || !img.IsImage() {
		t.Fatalf("img=%+v ok=%v", img, ok)
	}
	got := EstimateCostUSD(usage.Event{
		ProviderType: "openai", TargetModel: "dall-e-3",
		Modality: "image",
		Metrics:  map[string]any{"images": 2.0},
	})
	// 2 * 0.04 = 0.08
	if got == nil || *got < 0.079 || *got > 0.081 {
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
