import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { UsersIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import {
	type OrgMember,
	type OrgRole,
	orgMembersQueryOptions,
	updateOrgMemberRoleMutationOptions,
} from "#/api/organization";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
	Empty,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/users")({
	staticData: {
		getTitle: () => "Users",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const members = useQuery(orgMembersQueryOptions(orgId));
	const updateRole = useMutation(updateOrgMemberRoleMutationOptions());
	const [busyUserId, setBusyUserId] = useState<string | null>(null);

	const me = useMemo(
		() => (members.data ?? []).find((m) => m.user_id === user?.id),
		[members.data, user?.id],
	);
	const isOwner = me?.role === "owner";
	const isOrgAdmin = isOwner || me?.role === "admin";

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
				description="Organization members. Owners and admins manage service-account keys and quotas; members manage their personal keys. Only the owner can change roles."
				actions={
					isOrgAdmin ? (
						<Button nativeButton={false} render={<Link to="/app/quotas" />}>
							Manage quotas
						</Button>
					) : null
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
								Invite members from Organizations.
							</EmptyDescription>
						</EmptyHeader>
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
		</PageBody>
	);
}
