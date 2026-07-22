import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { PlusIcon, UsersIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import {
	inviteOrgMemberMutationOptions,
	type OrgMember,
	type OrgRole,
	orgInvitesQueryOptions,
	orgMembersQueryOptions,
	resendOrgInviteMutationOptions,
	revokeOrgInviteMutationOptions,
	updateOrgMemberRoleMutationOptions,
} from "#/api/organization";
import { InfoAlert } from "#/components/info-alert";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
	Empty,
	EmptyContent,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import {
	Sheet,
	SheetContent,
	SheetDescription,
	SheetFooter,
	SheetHeader,
	SheetTitle,
} from "#/components/ui/sheet";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/users")({
	...pageTitle("Users"),
	component: RouteComponent,
});

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const members = useQuery(orgMembersQueryOptions(orgId));
	const invites = useQuery({
		...orgInvitesQueryOptions(orgId),
		enabled: !!orgId,
	});
	const updateRole = useMutation(updateOrgMemberRoleMutationOptions());
	const revokeInvite = useMutation(revokeOrgInviteMutationOptions());
	const resendInvite = useMutation(resendOrgInviteMutationOptions());
	const [busyUserId, setBusyUserId] = useState<string | null>(null);
	const [inviteOpen, setInviteOpen] = useState(false);
	const [email, setEmail] = useState("");
	const [inviteError, setInviteError] = useState<string | null>(null);

	const me = useMemo(
		() => (members.data ?? []).find((m) => m.user_id === user?.id),
		[members.data, user?.id],
	);
	const isOwner = me?.role === "owner";
	const isOrgAdmin = isOwner || me?.role === "admin";

	const pendingInvites = useMemo(
		() => (invites.data ?? []).filter((i) => i.status === "pending"),
		[invites.data],
	);

	const invite = useMutation({
		...inviteOrgMemberMutationOptions(),
		onSuccess: (outcome) => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "members"],
			});
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "invites"],
			});
			setEmail("");
			setInviteOpen(false);
			toast.success(
				outcome.status === "invited"
					? "Invite sent"
					: "Member added and notified",
			);
		},
	});

	async function applyRole(m: OrgMember, role: OrgRole) {
		if (role === m.role) return;
		if (role === "owner") {
			const ok = window.confirm(
				`Transfer ownership to ${m.email}? You will become an admin.`,
			);
			if (!ok) return;
		}
		setBusyUserId(m.user_id);
		try {
			await updateRole.mutateAsync({ orgId, userId: m.user_id, role });
			await qc.invalidateQueries({
				queryKey: ["organizations", orgId, "members"],
			});
			toast.success(
				role === "owner" ? "Ownership transferred" : `Role set to ${role}`,
			);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : "Failed to update role");
		} finally {
			setBusyUserId(null);
		}
	}

	return (
		<PageBody>
			<PageHeader
				title="Users"
				description={org ? `People in ${org.name}.` : "Organization members."}
				info="Invite by email — existing users are added and emailed; new users get an accept link."
				actions={
					<div className="flex flex-wrap gap-2">
						{isOrgAdmin ? (
							<Button onClick={() => setInviteOpen(true)} disabled={!orgId}>
								<PlusIcon />
								Invite member
							</Button>
						) : null}
						{isOrgAdmin ? (
							<Button
								variant="outline"
								nativeButton={false}
								render={<Link to="/app/quotas" />}
							>
								Manage quotas
							</Button>
						) : null}
					</div>
				}
			/>
			<QueryGate
				isPending={members.isPending}
				isError={members.isError}
				error={members.error}
				onRetry={() => members.refetch()}
			>
				{(members.data ?? []).length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<UsersIcon />
							</EmptyMedia>
							<EmptyTitle>No members</EmptyTitle>
							<EmptyDescription>
								Invite people by email. Existing users are added immediately;
								new users receive an invite link to create their account.
							</EmptyDescription>
						</EmptyHeader>
						{isOrgAdmin ? (
							<EmptyContent>
								<Button onClick={() => setInviteOpen(true)} disabled={!orgId}>
									<PlusIcon />
									Invite member
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Email</TableHead>
								<TableHead>Org role</TableHead>
								<TableHead className="w-44" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{(members.data ?? []).map((m) => (
								<TableRow key={m.user_id}>
									<TableCell className="font-medium">{m.name}</TableCell>
									<TableCell>{m.email}</TableCell>
									<TableCell>
										{isOwner ? (
											<Select
												value={m.role}
												disabled={busyUserId === m.user_id}
												onValueChange={(v) => {
													const role = (v ?? m.role) as OrgRole;
													void applyRole(m, role);
												}}
											>
												<SelectTrigger className="w-40">
													<SelectValue />
												</SelectTrigger>
												<SelectContent>
													<SelectItem value="member">member</SelectItem>
													<SelectItem value="admin">admin</SelectItem>
													<SelectItem value="owner">
														owner (transfer)
													</SelectItem>
												</SelectContent>
											</Select>
										) : (
											<Badge variant="secondary">{m.role}</Badge>
										)}
									</TableCell>
									<TableCell>
										{isOrgAdmin ? (
											<Button
												variant="outline"
												size="sm"
												nativeButton={false}
												render={<Link to="/app/quotas" />}
											>
												Set user quota
											</Button>
										) : null}
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				)}
			</QueryGate>

			{isOrgAdmin && pendingInvites.length > 0 ? (
				<section className="mt-8 space-y-3">
					<h2 className="text-sm font-medium">Pending invites</h2>
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Email</TableHead>
								<TableHead>Expires</TableHead>
								<TableHead className="w-48" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{pendingInvites.map((inv) => (
								<TableRow key={inv.id}>
									<TableCell>{inv.email}</TableCell>
									<TableCell className="text-muted-foreground text-sm">
										{new Date(inv.expires_at).toLocaleString()}
									</TableCell>
									<TableCell className="flex gap-2">
										<Button
											variant="outline"
											size="sm"
											disabled={resendInvite.isPending}
											onClick={() =>
												resendInvite.mutate(
													{ orgId, inviteId: inv.id },
													{
														onSuccess: () => toast.success("Invite resent"),
														onError: (err) =>
															toast.error(
																err.message || "Failed to resend invite",
															),
													},
												)
											}
										>
											Resend
										</Button>
										<Button
											variant="ghost"
											size="sm"
											disabled={revokeInvite.isPending}
											onClick={() =>
												revokeInvite.mutate(
													{ orgId, inviteId: inv.id },
													{
														onSuccess: () => {
															void qc.invalidateQueries({
																queryKey: ["organizations", orgId, "invites"],
															});
															toast.success("Invite revoked");
														},
														onError: (err) =>
															toast.error(
																err.message || "Failed to revoke invite",
															),
													},
												)
											}
										>
											Revoke
										</Button>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</section>
			) : null}

			<Sheet open={inviteOpen} onOpenChange={setInviteOpen}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Invite member</SheetTitle>
						<SheetDescription>
							Enter an email address to invite.
						</SheetDescription>
						<InfoAlert>
							Existing users are added and emailed; new users receive an invite
							link to set a password and join.
						</InfoAlert>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!orgId) return;
							setInviteError(null);
							invite.mutate(
								{ orgId, email },
								{
									onError: (err) =>
										setInviteError(
											err instanceof Error ? err.message : "Invite failed",
										),
								},
							);
						}}
					>
						<div className="space-y-1">
							<Label htmlFor="member-email">Email</Label>
							<Input
								id="member-email"
								type="email"
								value={email}
								onChange={(e) => setEmail(e.target.value)}
								required
							/>
						</div>
						{inviteError ? (
							<p className="text-destructive text-xs">{inviteError}</p>
						) : null}
						<SheetFooter>
							<Button
								type="button"
								variant="outline"
								onClick={() => setInviteOpen(false)}
							>
								Cancel
							</Button>
							<Button
								type="submit"
								disabled={invite.isPending || !email.trim() || !orgId}
							>
								{invite.isPending ? "Sending…" : "Send invite"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>
		</PageBody>
	);
}
