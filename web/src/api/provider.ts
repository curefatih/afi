import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type ProviderCapabilities = {
	chat: boolean;
	stream: boolean;
	tts?: boolean;
	stt?: boolean;
};

export type Provider = {
	id: string;
	organization_id: string;
	name: string;
	type: string;
	base_url: string;
	api_key_env: string;
	capabilities: ProviderCapabilities;
	created_at: string;
};

export const PROVIDER_TYPE_PRESETS: Record<
	string,
	{ name: string; base_url: string; api_key_env: string; caps: ProviderCapabilities }
> = {
	openai: {
		name: "OpenAI",
		base_url: "https://api.openai.com/v1",
		api_key_env: "OPENAI_API_KEY",
		caps: { chat: true, stream: true, tts: true, stt: true },
	},
	anthropic: {
		name: "Anthropic",
		base_url: "https://api.anthropic.com/v1",
		api_key_env: "ANTHROPIC_API_KEY",
		caps: { chat: true, stream: true, tts: false, stt: false },
	},
	gemini: {
		name: "Gemini",
		base_url: "https://generativelanguage.googleapis.com/v1beta",
		api_key_env: "GEMINI_API_KEY",
		caps: { chat: true, stream: true, tts: false, stt: false },
	},
	openai_compatible: {
		name: "Ollama / compatible",
		base_url: "http://127.0.0.1:11434/v1",
		api_key_env: "OLLAMA_API_KEY",
		caps: { chat: true, stream: true, tts: true, stt: true },
	},
};

export const providersQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "providers"],
		queryFn: () =>
			apiFetch<Provider[]>(
				`/api/v1/platform/organizations/${orgId}/providers`,
			),
		enabled: !!orgId,
	});

export type CreateProviderInput = {
	orgId: string;
	name: string;
	type?: string;
	base_url: string;
	api_key_env?: string;
	capabilities?: ProviderCapabilities;
};

export const createProviderMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateProviderInput) =>
			apiFetch<Provider>(
				`/api/v1/platform/organizations/${orgId}/providers`,
				{ method: "POST", body },
			),
	});

export type UpdateProviderInput = {
	providerId: string;
	name: string;
	base_url: string;
	api_key_env: string;
};

export const updateProviderMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ providerId, ...body }: UpdateProviderInput) =>
			apiFetch<Provider>(`/api/v1/platform/providers/${providerId}`, {
				method: "PATCH",
				body,
			}),
	});

export const deleteProviderMutationOptions = () =>
	mutationOptions({
		mutationFn: (providerId: string) =>
			apiFetch<void>(`/api/v1/platform/providers/${providerId}`, {
				method: "DELETE",
			}),
	});
