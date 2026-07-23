/** Shape returned by gateway GET /v1/models (catalog-enriched). */
export type GatewayModel = {
	id: string;
	mode?: string;
	supports_streaming?: boolean;
	supports_tts?: boolean;
	supports_stt?: boolean;
	supports_embedding?: boolean;
	capabilities?: {
		chat?: boolean;
		stream?: boolean;
		tts?: boolean;
		stt?: boolean;
		embedding?: boolean;
	};
};

export function isChatModel(m: GatewayModel): boolean {
	if (!m.id) return false;
	if (m.capabilities?.chat === false) return false;
	if (m.supports_tts || m.supports_stt || m.supports_embedding) return false;
	if (m.mode && m.mode !== "chat") return false;
	return true;
}

export function isTTSModel(m: GatewayModel): boolean {
	if (!m.id) return false;
	if (m.mode === "audio_speech") return true;
	return Boolean(m.supports_tts || m.capabilities?.tts);
}

export function isSTTModel(m: GatewayModel): boolean {
	if (!m.id) return false;
	if (m.mode === "audio_transcription") return true;
	return Boolean(m.supports_stt || m.capabilities?.stt);
}

export function isEmbeddingModel(m: GatewayModel): boolean {
	if (!m.id) return false;
	if (m.mode === "embedding") return true;
	return Boolean(m.supports_embedding || m.capabilities?.embedding);
}
