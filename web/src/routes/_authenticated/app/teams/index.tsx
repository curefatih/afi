import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { PlusIcon, Users2Icon } from "lucide-react";
import { useMemo, useState } from "react";
import { orgMembersQueryOptions } from "#/api/organization";
import { teamsQueryOptions } from "#/api/team";
import { CreateTeamSheet } from "#/components/create-team-sheet";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import TeamCard from "#/components/team-card";
import { Button } from "#/components/ui/button";
import {
	Empty,
	EmptyContent,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/teams/")({
	...pageTitle("Teams"),
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const user = useAuthUser();
	const [open, setOpen] = useState(false);

	const teamsQuery = useQuery({
		...teamsQueryOptions(orgId),
	});
	const members = useQuery(orgMembersQueryOptions(orgId));

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const teams = teamsQuery.data ?? activeOrg?.teams ?? [];

	return (
		<PageBody>
			<PageHeader
				title="Teams"
				description="Teams group members and own projects within the organization."
				actions={
					isOrgAdmin ? (
						<Button onClick={() => setOpen(true)} disabled={!orgId}>
							<PlusIcon />
							New team
						</Button>
					) : null
				}
			/>

			<QueryGate
				isPending={(teamsQuery.isPending && !teams.length) || members.isPending}
				isError={teamsQuery.isError}
				error={teamsQuery.error}
				onRetry={() => void teamsQuery.refetch()}
			>
				{teams.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<Users2Icon />
							</EmptyMedia>
							<EmptyTitle>No teams</EmptyTitle>
							<EmptyDescription>
								{isOrgAdmin
									? "This organization has no teams yet."
									: "You are not assigned to any teams yet. Ask an organization admin or team owner to add you."}
							</EmptyDescription>
						</EmptyHeader>
						{isOrgAdmin ? (
							<EmptyContent>
								<Button onClick={() => setOpen(true)} disabled={!orgId}>
									<PlusIcon />
									Create team
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<div className="flex flex-wrap gap-4">
						{teams.map((team) => (
							<TeamCard
								key={team.id}
								id={team.id}
								name={team.name}
								description={team.team_id}
								previewMembers={[]}
								memberCount={0}
								tags={[]}
							/>
						))}
					</div>
				)}
			</QueryGate>

			<CreateTeamSheet open={open} onOpenChange={setOpen} />
		</PageBody>
	);
}
