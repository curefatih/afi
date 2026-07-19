import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { UsersIcon } from "lucide-react";
import { useMemo } from "react";
import { orgMembersQueryOptions } from "#/api/organization";
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
	const members = useQuery(orgMembersQueryOptions(orgId));

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	return (
		<PageBody>
			<PageHeader
				title="Users"
				description="Organization members. Admins can set per-user quotas from Quotas."
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
								<TableHead className="w-40" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{(members.data ?? []).map((m) => (
								<TableRow key={m.user_id}>
									<TableCell className="font-medium">{m.name}</TableCell>
									<TableCell>{m.email}</TableCell>
									<TableCell>
										<Badge variant="secondary">{m.role}</Badge>
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
