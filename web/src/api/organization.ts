import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";
import type { Organization, Project, Team } from "#/state/organization-state";

export type OrgSummary = {
	id: string;
	name: string;
	created_at?: string;
};

export type OrgMember = {
	user_id: string;
	email: string;
	name: string;
	role: string;
};

export const organizationsQueryOptions = () =>
	queryOptions({
		queryKey: ["organizations"],
		queryFn: () => apiFetch<OrgSummary[]>("/api/v1/platform/organizations"),
	});

export const orgMembersQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "members"],
		queryFn: () =>
			apiFetch<OrgMember[]>(`/api/v1/platform/organizations/${orgId}/members`),
		enabled: !!orgId,
	});

export const createOrganizationMutationOptions = () =>
	mutationOptions({
		mutationFn: (body: { name: string }) =>
			apiFetch<OrgSummary>("/api/v1/platform/organizations", {
				method: "POST",
				body,
			}),
	});

export type OrgInvite = {
	id: string;
	organization_id: string;
	email: string;
	role: string;
	invited_by_user_id: string;
	status: string;
	expires_at: string;
	created_at: string;
	accepted_at?: string | null;
};

export type InviteOutcome = {
	status: "added" | "invited";
	member?: OrgMember;
	invite?: OrgInvite;
};

export const inviteOrgMemberMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, email }: { orgId: string; email: string }) =>
			apiFetch<InviteOutcome>(
				`/api/v1/platform/organizations/${orgId}/members`,
				{
					method: "POST",
					body: { email },
				},
			),
	});

/** @deprecated use inviteOrgMemberMutationOptions */
export const addOrgMemberMutationOptions = inviteOrgMemberMutationOptions;

export const orgInvitesQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "invites"],
		queryFn: () =>
			apiFetch<OrgInvite[]>(`/api/v1/platform/organizations/${orgId}/invites`),
		enabled: !!orgId,
	});

export const revokeOrgInviteMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, inviteId }: { orgId: string; inviteId: string }) =>
			apiFetch<void>(
				`/api/v1/platform/organizations/${orgId}/invites/${inviteId}`,
				{ method: "DELETE" },
			),
	});

export const resendOrgInviteMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, inviteId }: { orgId: string; inviteId: string }) =>
			apiFetch<OrgInvite>(
				`/api/v1/platform/organizations/${orgId}/invites/${inviteId}/resend`,
				{ method: "POST" },
			),
	});

export type OrgMailSettings = {
	selected: string;
	default_provider: string;
	enabled_providers: string[];
	from?: string;
	public_app_url?: string;
};

export const orgMailQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "mail"],
		queryFn: () =>
			apiFetch<OrgMailSettings>(
				`/api/v1/platform/organizations/${orgId}/mail`,
			),
		enabled: !!orgId,
	});

export const updateOrgMailMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			orgId,
			provider,
		}: {
			orgId: string;
			provider: string;
		}) =>
			apiFetch<OrgMailSettings>(
				`/api/v1/platform/organizations/${orgId}/mail`,
				{ method: "PATCH", body: { provider } },
			),
	});

export const testOrgMailMutationOptions = () =>
	mutationOptions({
		mutationFn: (orgId: string) =>
			apiFetch<{ status: string; to: string }>(
				`/api/v1/platform/organizations/${orgId}/mail/test`,
				{ method: "POST" },
			),
	});

export type InvitePreview = {
	email: string;
	organization_id: string;
	organization_name: string;
	expires_at: string;
	user_exists: boolean;
};

export const invitePreviewQueryOptions = (token: string) =>
	queryOptions({
		queryKey: ["auth", "invites", token],
		queryFn: () =>
			apiFetch<InvitePreview>(`/api/v1/platform/auth/invites/${token}`, {
				auth: false,
			}),
		enabled: !!token,
	});

export const acceptInviteMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			token,
			name,
			password,
		}: {
			token: string;
			name?: string;
			password?: string;
		}) =>
			apiFetch<{
				member: OrgMember;
				user: { id: string; email: string; name: string; role: string };
				token: string;
			}>(`/api/v1/platform/auth/invites/${token}/accept`, {
				method: "POST",
				auth: false,
				body: { name, password },
			}),
	});

export type OrgRole = "owner" | "admin" | "member";

export const updateOrgMemberRoleMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			orgId,
			userId,
			role,
		}: {
			orgId: string;
			userId: string;
			role: OrgRole;
		}) =>
			apiFetch<OrgMember>(
				`/api/v1/platform/organizations/${orgId}/members/${userId}`,
				{ method: "PATCH", body: { role } },
			),
	});

export const orgTeamsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "teams"],
		queryFn: () =>
			apiFetch<Team[]>(`/api/v1/platform/organizations/${orgId}/teams`),
		enabled: !!orgId,
	});

export const orgProjectsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "projects"],
		queryFn: () =>
			apiFetch<Project[]>(`/api/v1/platform/organizations/${orgId}/projects`),
		enabled: !!orgId,
	});

export function toOrganization(
	org: OrgSummary,
	teams: Team[] = [],
	projects: Project[] = [],
): Organization {
	return {
		id: org.id,
		name: org.name,
		created_at: org.created_at,
		teams,
		projects,
	};
}
