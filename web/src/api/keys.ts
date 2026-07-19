import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type ApiKey = {
	id: string;
	project_id: string;
	organization_id: string;
	name: string;
	key_prefix: string;
	/** Present only on create responses. */
	key?: string;
	created_at: string;
};

export const projectKeysQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "keys"],
		queryFn: () =>
			apiFetch<ApiKey[]>(`/api/v1/platform/projects/${projectId}/keys`),
		enabled: !!projectId,
	});

export type CreateKeyInput = {
	projectId: string;
	name: string;
	key?: string;
};

export const createKeyMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ projectId, name, key }: CreateKeyInput) =>
			apiFetch<ApiKey>(`/api/v1/platform/projects/${projectId}/keys`, {
				method: "POST",
				body: { name, key },
			}),
	});
