import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type CredentialStorageKind = "env" | "encrypted_db" | "vault";

export type Credential = {
	id: string;
	organization_id: string;
	name: string;
	provider_type: string;
	storage_kind: CredentialStorageKind;
	secret_ref?: string;
	key_version?: number;
	status: "active" | "disabled";
	has_secret: boolean;
	created_at: string;
	updated_at: string;
};

export type CredentialAssignment = {
	id: string;
	credential_id: string;
	organization_id: string;
	provider_type: string;
	scope_type: "organization" | "project" | "api_key";
	scope_id: string;
	created_at: string;
	created_by?: string;
};

export const credentialsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "credentials"],
		queryFn: () =>
			apiFetch<Credential[]>(
				`/api/v1/platform/organizations/${orgId}/credentials`,
			),
		enabled: !!orgId,
	});

export const credentialAssignmentsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "credential-assignments"],
		queryFn: () =>
			apiFetch<CredentialAssignment[]>(
				`/api/v1/platform/organizations/${orgId}/credential-assignments`,
			),
		enabled: !!orgId,
	});

export type CreateCredentialInput = {
	orgId: string;
	name: string;
	provider_type: string;
	storage_kind: CredentialStorageKind;
	secret_ref?: string;
	secret_value?: string;
};

export const createCredentialMutationOptions = () =>
	mutationOptions({
		mutationFn: (input: CreateCredentialInput) =>
			apiFetch<Credential>(
				`/api/v1/platform/organizations/${input.orgId}/credentials`,
				{
					method: "POST",
					body: {
						name: input.name,
						provider_type: input.provider_type,
						storage_kind: input.storage_kind,
						secret_ref: input.secret_ref,
						secret_value: input.secret_value,
					},
				},
			),
	});

export const deleteCredentialMutationOptions = () =>
	mutationOptions({
		mutationFn: (credentialId: string) =>
			apiFetch<void>(`/api/v1/platform/credentials/${credentialId}`, {
				method: "DELETE",
			}),
	});

export type AssignCredentialInput = {
	orgId: string;
	credential_id: string;
	scope_type: "organization" | "project" | "api_key";
	scope_id: string;
};

export const assignCredentialMutationOptions = () =>
	mutationOptions({
		mutationFn: (input: AssignCredentialInput) =>
			apiFetch<CredentialAssignment>(
				`/api/v1/platform/organizations/${input.orgId}/credential-assignments`,
				{
					method: "PUT",
					body: {
						credential_id: input.credential_id,
						scope_type: input.scope_type,
						scope_id: input.scope_id,
					},
				},
			),
	});

export const deleteCredentialAssignmentMutationOptions = () =>
	mutationOptions({
		mutationFn: (assignmentId: string) =>
			apiFetch<void>(
				`/api/v1/platform/credential-assignments/${assignmentId}`,
				{ method: "DELETE" },
			),
	});
