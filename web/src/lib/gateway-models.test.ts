import { describe, expect, it } from "vitest";
import { isChatModel, isSTTModel, isTTSModel } from "./gateway-models";

describe("gateway model classifiers", () => {
	it("treats plain chat models as chat", () => {
		expect(isChatModel({ id: "gpt-4o" })).toBe(true);
		expect(isChatModel({ id: "gpt-4o", mode: "chat" })).toBe(true);
		expect(isChatModel({ id: "gpt-4o", capabilities: { chat: true } })).toBe(
			true,
		);
	});

	it("excludes audio and non-chat modes from chat", () => {
		expect(isChatModel({ id: "" })).toBe(false);
		expect(isChatModel({ id: "tts-1", supports_tts: true })).toBe(false);
		expect(isChatModel({ id: "whisper", supports_stt: true })).toBe(false);
		expect(isChatModel({ id: "tts-1", mode: "audio_speech" })).toBe(false);
		expect(isChatModel({ id: "blocked", capabilities: { chat: false } })).toBe(
			false,
		);
	});

	it("detects TTS models", () => {
		expect(isTTSModel({ id: "tts-1", mode: "audio_speech" })).toBe(true);
		expect(isTTSModel({ id: "tts-1", supports_tts: true })).toBe(true);
		expect(isTTSModel({ id: "tts-1", capabilities: { tts: true } })).toBe(true);
		expect(isTTSModel({ id: "gpt-4o" })).toBe(false);
	});

	it("detects STT models", () => {
		expect(isSTTModel({ id: "whisper-1", mode: "audio_transcription" })).toBe(
			true,
		);
		expect(isSTTModel({ id: "whisper-1", supports_stt: true })).toBe(true);
		expect(isSTTModel({ id: "whisper-1", capabilities: { stt: true } })).toBe(
			true,
		);
		expect(isSTTModel({ id: "gpt-4o" })).toBe(false);
	});
});
