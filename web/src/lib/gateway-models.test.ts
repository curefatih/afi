import { describe, expect, it } from "vitest";
import { isChatModel, isSTTModel, isTTSModel } from "./gateway-models";
import {
	appendSTTFormFields,
	buildTTSRequestBody,
	defaultVoiceForModel,
	parseExtraConfig,
	pickPreferredModel,
	voicePresetsForModel,
} from "./playground-audio";

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
		expect(
			isTTSModel({
				id: "eleven-tts",
				mode: "audio_speech",
				provider_type: "elevenlabs",
				supports_tts: true,
			}),
		).toBe(true);
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
		expect(
			isSTTModel({
				id: "eleven-stt",
				mode: "audio_transcription",
				provider_type: "elevenlabs",
				supports_stt: true,
			}),
		).toBe(true);
		expect(isSTTModel({ id: "gpt-4o" })).toBe(false);
	});
});

describe("playground audio config", () => {
	it("prefers seeded model ids when present", () => {
		expect(
			pickPreferredModel(
				[
					{ id: "custom-tts", mode: "audio_speech" },
					{ id: "tts-1", mode: "audio_speech" },
				],
				["tts-1"],
			),
		).toBe("tts-1");
		expect(
			pickPreferredModel([{ id: "custom-tts", mode: "audio_speech" }], [
				"tts-1",
			]),
		).toBe("custom-tts");
	});

	it("offers voice presets only for known providers", () => {
		expect(
			voicePresetsForModel({ id: "x", provider_type: "openai" }).some(
				(v) => v.id === "alloy",
			),
		).toBe(true);
		expect(
			defaultVoiceForModel({ id: "x", provider_type: "elevenlabs" }),
		).toBe("21m00Tcm4TlvDq8ikWAM");
		expect(voicePresetsForModel({ id: "x", provider_type: "custom" })).toEqual(
			[],
		);
		expect(defaultVoiceForModel({ id: "x", provider_type: "custom" })).toBe(
			"alloy",
		);
	});

	it("parses extra JSON objects", () => {
		expect(parseExtraConfig("")).toEqual({ ok: true, value: {} });
		expect(parseExtraConfig('{"a":1}')).toEqual({
			ok: true,
			value: { a: 1 },
		});
		expect(parseExtraConfig("[]").ok).toBe(false);
		expect(parseExtraConfig("{").ok).toBe(false);
	});

	it("builds TTS body with form fields winning over extras", () => {
		const built = buildTTSRequestBody({
			model: "tts-1",
			input: "hi",
			voice: "nova",
			responseFormat: "mp3",
			speed: "1.25",
			extraJSON: '{"voice":"ignored","instructions":"soft"}',
		});
		expect(built.ok).toBe(true);
		if (!built.ok) return;
		expect(built.body).toEqual({
			instructions: "soft",
			model: "tts-1",
			input: "hi",
			voice: "nova",
			response_format: "mp3",
			speed: 1.25,
		});
	});

	it("omits empty optional TTS fields", () => {
		const built = buildTTSRequestBody({
			model: "m",
			input: "hi",
			voice: "",
			responseFormat: "",
			speed: "",
			extraJSON: "{}",
		});
		expect(built.ok).toBe(true);
		if (!built.ok) return;
		expect(built.body).toEqual({ model: "m", input: "hi" });
	});

	it("appends STT form fields with form winning over extras", () => {
		const form = new FormData();
		form.set("file", new Blob(["x"]), "a.wav");
		const result = appendSTTFormFields(form, {
			model: "whisper-1",
			language: "en",
			prompt: "names",
			responseFormat: "json",
			extraJSON: '{"language":"fr","temperature":0}',
		});
		expect(result.ok).toBe(true);
		expect(form.get("model")).toBe("whisper-1");
		expect(form.get("language")).toBe("en");
		expect(form.get("prompt")).toBe("names");
		expect(form.get("response_format")).toBe("json");
		expect(form.get("temperature")).toBe("0");
	});
});
