import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type KeyKind = "personal" | "service_account";

export type ApiKey = {
	id: string;
	project_id?: string;
	organization_id: string;
	name: string;
	kind: KeyKind;
	owner_user_id?: string;
	key_prefix: string;
	/** Present only on create responses. */
	key?: string;
	created_at: string;
};

export const orgKeysQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "keys"],
		queryFn: () =>
			apiFetch<ApiKey[]>(`/api/v1/platform/organizations/${orgId}/keys`),
		enabled: !!orgId,
	});

export const projectKeysQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "keys"],
		queryFn: () =>
			apiFetch<ApiKey[]>(`/api/v1/platform/projects/${projectId}/keys`),
		enabled: !!projectId,
	});

export type CreateOrgKeyInput = {
	orgId: string;
	name: string;
	kind: KeyKind;
	project_id?: string;
	key?: string;
};

export const createOrgKeyMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateOrgKeyInput) =>
			apiFetch<ApiKey>(`/api/v1/platform/organizations/${orgId}/keys`, {
				method: "POST",
				body,
			}),
	});

/** Project detail: creates a project-scoped service account key (admin only). */
export type CreateProjectKeyInput = {
	projectId: string;
	name: string;
	key?: string;
};

export const createKeyMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ projectId, name, key }: CreateProjectKeyInput) =>
			apiFetch<ApiKey>(`/api/v1/platform/projects/${projectId}/keys`, {
				method: "POST",
				body: { name, key },
			}),
	});

export const deleteKeyMutationOptions = () =>
	mutationOptions({
		mutationFn: (keyId: string) =>
			apiFetch<void>(`/api/v1/platform/keys/${keyId}`, {
				method: "DELETE",
			}),
	});
