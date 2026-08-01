import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type ControlPlanePeer = {
	id: string;
	name: string;
	region_id: string;
	base_url?: string;
	status: string;
	last_sync_at?: string | null;
	last_sync_cursor: number;
	last_sync_error?: string;
	created_at: string;
	updated_at: string;
};

export type FederationPeerWithToken = {
	peer: ControlPlanePeer;
	join_token: string;
};

export const federationPeersQueryOptions = () =>
	queryOptions({
		queryKey: ["federation", "peers"],
		queryFn: () =>
			apiFetch<ControlPlanePeer[]>("/api/v1/platform/federation/peers"),
	});

export const registerFederationPeerMutationOptions = () =>
	mutationOptions({
		mutationFn: (body: {
			name: string;
			region_id: string;
			base_url?: string;
		}) =>
			apiFetch<FederationPeerWithToken>("/api/v1/platform/federation/peers", {
				method: "POST",
				body,
			}),
	});

export const updateFederationPeerMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			peerId,
			...body
		}: {
			peerId: string;
			name?: string;
			base_url?: string;
			status?: string;
		}) =>
			apiFetch<ControlPlanePeer>(
				`/api/v1/platform/federation/peers/${peerId}`,
				{ method: "PATCH", body },
			),
	});

export const rotateFederationPeerTokenMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ peerId }: { peerId: string }) =>
			apiFetch<FederationPeerWithToken>(
				`/api/v1/platform/federation/peers/${peerId}/rotate-join-token`,
				{ method: "POST" },
			),
	});
