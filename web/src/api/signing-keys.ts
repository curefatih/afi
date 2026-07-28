import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type SigningKeyStatus = "active" | "disabled";

export type SigningKey = {
	id: string;
	key_id: string;
	project_id?: string;
	organization_id: string;
	environment_id?: string;
	name: string;
	algorithm: string;
	public_key_pem: string;
	status: SigningKeyStatus;
	created_at: string;
	updated_at: string;
};

export const orgSigningKeysQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "signing-keys"],
		queryFn: () =>
			apiFetch<SigningKey[]>(
				`/api/v1/platform/organizations/${orgId}/signing-keys`,
			),
		enabled: !!orgId,
	});

export type CreateSigningKeyInput = {
	orgId: string;
	key_id: string;
	name: string;
	algorithm?: string;
	public_key_pem: string;
	project_id?: string;
	environment_id?: string;
};

export const createSigningKeyMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateSigningKeyInput) =>
			apiFetch<SigningKey>(
				`/api/v1/platform/organizations/${orgId}/signing-keys`,
				{
					method: "POST",
					body: {
						key_id: body.key_id,
						name: body.name,
						algorithm: body.algorithm || "ed25519",
						public_key_pem: body.public_key_pem,
						project_id: body.project_id,
						environment_id: body.environment_id,
					},
				},
			),
	});

export type UpdateSigningKeyInput = {
	signingKeyId: string;
	name?: string;
	status?: SigningKeyStatus;
};

export const updateSigningKeyMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ signingKeyId, ...body }: UpdateSigningKeyInput) =>
			apiFetch<SigningKey>(`/api/v1/platform/signing-keys/${signingKeyId}`, {
				method: "PATCH",
				body,
			}),
	});

export type RotateSigningKeyInput = {
	signingKeyId: string;
	public_key_pem: string;
};

export const rotateSigningKeyMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ signingKeyId, public_key_pem }: RotateSigningKeyInput) =>
			apiFetch<SigningKey>(
				`/api/v1/platform/signing-keys/${signingKeyId}/rotate`,
				{
					method: "POST",
					body: { public_key_pem },
				},
			),
	});

export const deleteSigningKeyMutationOptions = () =>
	mutationOptions({
		mutationFn: (signingKeyId: string) =>
			apiFetch<void>(`/api/v1/platform/signing-keys/${signingKeyId}`, {
				method: "DELETE",
			}),
	});
