/** Data-plane (gateway) base URL for inference. */
export const GATEWAY_API_URL =
	import.meta.env.VITE_GATEWAY_API_URL ?? "http://localhost:8080";

/** Local-dev virtual API key (seeded by control plane). */
export const GATEWAY_API_KEY =
	import.meta.env.VITE_GATEWAY_API_KEY ?? "sk-project-local-dev-token-12345";
