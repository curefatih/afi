import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Users2Icon } from "lucide-react";
import { teamsQueryOptions } from "#/api/team";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import TeamCard from "#/components/team-card";
import {
	Empty,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/teams/")({
	staticData: {
		getTitle: () => "Teams",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";

	const teamsQuery = useQuery({
		...teamsQueryOptions(orgId),
	});

	const teams = teamsQuery.data ?? activeOrg?.teams ?? [];

	return (
		<PageBody>
			<PageHeader
				title="Teams"
				description="Teams group members and own projects within the organization."
			/>

			<QueryGate
				isPending={teamsQuery.isPending && !teams.length}
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
								This organization has no teams yet.
							</EmptyDescription>
						</EmptyHeader>
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
		</PageBody>
	);
}
