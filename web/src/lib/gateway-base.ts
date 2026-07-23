/** Data-plane (gateway) base URL for inference. */
export const GATEWAY_API_URL =
	import.meta.env.VITE_GATEWAY_API_URL ?? "http://localhost:8080";

/** Local-dev virtual API key (seeded by control plane). */
export const GATEWAY_API_KEY =
	import.meta.env.VITE_GATEWAY_API_KEY ?? "sk-project-local-dev-token-12345";

/** Seeded local-dev project id (`SeedLocalDev`). */
export const SEEDED_PROJECT_ID = "proj_local";

/** Sentinel for playground "use seeded key" (same key as `SEEDED_PROJECT_ID`). */
export const PLAYGROUND_SEEDED_KEY = "seeded";

const playgroundKeyStoragePrefix = "afi-playground-api-key:";

function playgroundKeyStorageKey(projectId: string): string {
	return `${playgroundKeyStoragePrefix}${projectId}`;
}

/** Whether this selection should use the seeded local-dev virtual key. */
export function isSeededPlaygroundProject(projectId: string): boolean {
	return projectId === PLAYGROUND_SEEDED_KEY || projectId === SEEDED_PROJECT_ID;
}

/**
 * Resolve a cached playground virtual API key for a project.
 * Secrets are never user-pasted — only seeded or auto-provisioned values.
 */
export function resolvePlaygroundApiKey(projectId: string): string {
	if (isSeededPlaygroundProject(projectId)) {
		return GATEWAY_API_KEY;
	}
	if (typeof sessionStorage === "undefined") return "";
	return sessionStorage.getItem(playgroundKeyStorageKey(projectId)) ?? "";
}

/** Cache an auto-provisioned playground key for this browser tab. */
export function storePlaygroundApiKey(projectId: string, apiKey: string): void {
	if (isSeededPlaygroundProject(projectId)) return;
	if (typeof sessionStorage === "undefined") return;
	const trimmed = apiKey.trim();
	const key = playgroundKeyStorageKey(projectId);
	if (!trimmed) {
		sessionStorage.removeItem(key);
		return;
	}
	sessionStorage.setItem(key, trimmed);
}
