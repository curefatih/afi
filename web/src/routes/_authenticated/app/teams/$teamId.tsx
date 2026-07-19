import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Users2Icon } from "lucide-react";
import { teamMembersQueryOptions, teamQueryOptions } from "#/api/team";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";
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
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/teams/$teamId")({
	staticData: {
		getTitle: () => "Team",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const { teamId } = Route.useParams();
	const activeOrg = useActiveOrg();

	const teamQuery = useQuery({
		...teamQueryOptions(teamId),
	});

	const membersQuery = useQuery({
		...teamMembersQueryOptions(teamId),
	});

	const projects =
		activeOrg?.projects.filter((p) => p.team_id === teamId) ?? [];

	return (
		<PageBody>
			<QueryGate
				isPending={teamQuery.isPending}
				isError={teamQuery.isError}
				error={teamQuery.error}
				onRetry={() => void teamQuery.refetch()}
			>
				<PageHeader
					title={teamQuery.data?.name ?? "Team"}
					description="Members and projects for this team."
				/>

				<div className="grid gap-4 md:grid-cols-2">
					<Card>
						<CardHeader>
							<CardDescription>Team ID</CardDescription>
							<CardTitle className="font-mono text-sm break-all">
								{teamQuery.data?.id}
							</CardTitle>
						</CardHeader>
					</Card>
					<Card>
						<CardHeader>
							<CardDescription>Projects</CardDescription>
							<CardTitle className="text-base">
								{projects.length === 0
									? "None"
									: projects.map((p) => (
											<Link
												key={p.id}
												to="/app/projects/$projectId"
												params={{ projectId: p.id }}
												className="mr-2 hover:underline"
											>
												{p.name}
											</Link>
										))}
							</CardTitle>
						</CardHeader>
					</Card>
				</div>

				<Card>
					<CardHeader>
						<CardTitle>Members</CardTitle>
						<CardDescription>People with access to this team.</CardDescription>
					</CardHeader>
					<CardContent>
						<QueryGate
							isPending={membersQuery.isPending}
							isError={membersQuery.isError}
							error={membersQuery.error}
							onRetry={() => void membersQuery.refetch()}
						>
							{(membersQuery.data?.length ?? 0) === 0 ? (
								<Empty className="border min-h-40">
									<EmptyHeader>
										<EmptyMedia variant="icon">
											<Users2Icon />
										</EmptyMedia>
										<EmptyTitle>No members</EmptyTitle>
										<EmptyDescription>
											This team has no members listed.
										</EmptyDescription>
									</EmptyHeader>
								</Empty>
							) : (
								<Table>
									<TableHeader>
										<TableRow>
											<TableHead>Name</TableHead>
											<TableHead>Email</TableHead>
											<TableHead>Role</TableHead>
										</TableRow>
									</TableHeader>
									<TableBody>
										{membersQuery.data?.map((member) => (
											<TableRow key={member.user_id}>
												<TableCell className="font-medium">
													{member.name}
												</TableCell>
												<TableCell>{member.email}</TableCell>
												<TableCell>
													<Badge variant="secondary">{member.role}</Badge>
												</TableCell>
											</TableRow>
										))}
									</TableBody>
								</Table>
							)}
						</QueryGate>
					</CardContent>
				</Card>
			</QueryGate>
		</PageBody>
	);
}
