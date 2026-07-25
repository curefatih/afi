import type { GatewayModel } from "#/lib/gateway-models";

export type PlaygroundVoice = { id: string; label: string };

export const OPENAI_TTS_VOICES: readonly PlaygroundVoice[] = [
	{ id: "alloy", label: "alloy" },
	{ id: "echo", label: "echo" },
	{ id: "fable", label: "fable" },
	{ id: "onyx", label: "onyx" },
	{ id: "nova", label: "nova" },
	{ id: "shimmer", label: "shimmer" },
] as const;

export const ELEVENLABS_TTS_VOICES: readonly PlaygroundVoice[] = [
	{ id: "21m00Tcm4TlvDq8ikWAM", label: "Rachel" },
	{ id: "EXAVITQu4vr4xnSDxMaL", label: "Sarah" },
	{ id: "JBFqnCBsd6RMkjVDRZzb", label: "George" },
	{ id: "nPczCjzI2devNBz1zQrb", label: "Brian" },
	{ id: "FGY2WhTYpPnrIDTdsKH5", label: "Laura" },
	{ id: "TX3LPaxmHKxFdv7VOQHJ", label: "Liam" },
	{ id: "XB0fDUnXU5powFXDhCwa", label: "Charlotte" },
	{ id: "pFZP5JQG7iQjIQuC4Bku", label: "Lily" },
	{ id: "cgSgspJ2msm6clMCkdW9", label: "Jessica" },
	{ id: "cjVigY5qzO86Huf0OWal", label: "Eric" },
] as const;

export const TTS_RESPONSE_FORMATS = [
	"",
	"mp3",
	"opus",
	"aac",
	"flac",
	"wav",
	"pcm",
] as const;

export const STT_RESPONSE_FORMATS = [
	"",
	"json",
	"text",
	"srt",
	"verbose_json",
	"vtt",
] as const;

/** Soft-prefer known seed models, else first route. */
export function pickPreferredModel(
	list: GatewayModel[],
	preferred: string[],
): string {
	for (const id of preferred) {
		if (list.some((m) => m.id === id)) return id;
	}
	return list[0]?.id ?? "";
}

/** Optional voice presets for known providers; empty for unknown types. */
export function voicePresetsForModel(
	m: GatewayModel | undefined,
): readonly PlaygroundVoice[] {
	switch (m?.provider_type) {
		case "elevenlabs":
			return ELEVENLABS_TTS_VOICES;
		case "openai":
		case "openai_compatible":
			return OPENAI_TTS_VOICES;
		default:
			return [];
	}
}

export function defaultVoiceForModel(m: GatewayModel | undefined): string {
	const presets = voicePresetsForModel(m);
	if (presets[0]) return presets[0].id;
	// OpenAI-shaped default; harmless for providers that ignore voice.
	return "alloy";
}

export type ParseExtraResult =
	| { ok: true; value: Record<string, unknown> }
	| { ok: false; error: string };

/** Parse optional advanced JSON object (empty → {}). */
export function parseExtraConfig(raw: string): ParseExtraResult {
	const text = raw.trim();
	if (!text) return { ok: true, value: {} };
	try {
		const parsed: unknown = JSON.parse(text);
		if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
			return { ok: false, error: "Extra config must be a JSON object" };
		}
		return { ok: true, value: parsed as Record<string, unknown> };
	} catch (e) {
		return {
			ok: false,
			error: e instanceof Error ? e.message : "Invalid JSON",
		};
	}
}

export type TTSFormConfig = {
	model: string;
	input: string;
	voice: string;
	responseFormat: string;
	speed: string;
	extraJSON: string;
};

/** Build OpenAI-compatible speech body; form fields win over extras for core keys. */
export function buildTTSRequestBody(
	cfg: TTSFormConfig,
): { ok: true; body: Record<string, unknown> } | { ok: false; error: string } {
	const extra = parseExtraConfig(cfg.extraJSON);
	if (!extra.ok) return extra;

	const body: Record<string, unknown> = { ...extra.value };
	body.model = cfg.model;
	body.input = cfg.input;
	const voice = cfg.voice.trim();
	if (voice) body.voice = voice;
	else delete body.voice;

	const format = cfg.responseFormat.trim();
	if (format) body.response_format = format;
	else delete body.response_format;

	const speedRaw = cfg.speed.trim();
	if (speedRaw) {
		const speed = Number(speedRaw);
		if (!Number.isFinite(speed)) {
			return { ok: false, error: "Speed must be a number" };
		}
		body.speed = speed;
	} else {
		delete body.speed;
	}

	return { ok: true, body };
}

export type STTFormConfig = {
	model: string;
	language: string;
	prompt: string;
	responseFormat: string;
	extraJSON: string;
};

/** Append STT fields to FormData; form fields win over extras for core keys. */
export function appendSTTFormFields(
	form: FormData,
	cfg: STTFormConfig,
): { ok: true } | { ok: false; error: string } {
	const extra = parseExtraConfig(cfg.extraJSON);
	if (!extra.ok) return extra;

	for (const [k, v] of Object.entries(extra.value)) {
		if (v === undefined || v === null) continue;
		if (typeof v === "string" || typeof v === "number" || typeof v === "boolean") {
			form.set(k, String(v));
		} else {
			form.set(k, JSON.stringify(v));
		}
	}

	form.set("model", cfg.model);
	const language = cfg.language.trim();
	if (language) form.set("language", language);
	else form.delete("language");

	const prompt = cfg.prompt.trim();
	if (prompt) form.set("prompt", prompt);
	else form.delete("prompt");

	const format = cfg.responseFormat.trim();
	if (format) form.set("response_format", format);
	else form.delete("response_format");

	return { ok: true };
}

export const EMPTY_EXTRA_JSON = "{\n}\n";
