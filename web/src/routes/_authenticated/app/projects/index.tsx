import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { FolderKanbanIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { z } from "zod";
import { CreateProjectSheet } from "#/components/create-project-sheet";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";
import {
	Empty,
	EmptyContent,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";
import { Label } from "#/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { useOrgBootstrap } from "#/hooks/use-org-bootstrap";
import { pageTitle } from "#/lib/page-meta";
import { useActiveOrg } from "#/state/organization-state";

const projectsSearchSchema = z.object({
	team: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/app/projects/")({
	...pageTitle("Projects"),
	validateSearch: projectsSearchSchema,
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const navigate = useNavigate({ from: Route.fullPath });
	const { team: teamFilter } = Route.useSearch();
	const { isBootstrapping, isError, error, refetch, projectsQuery } =
		useOrgBootstrap();
	const [open, setOpen] = useState(false);

	const teams = activeOrg?.teams ?? [];
	const projects = activeOrg?.projects ?? [];
	const teamsById = useMemo(
		() => new Map(teams.map((team) => [team.id, team.name])),
		[teams],
	);

	const filteredProjects = useMemo(() => {
		if (!teamFilter) return projects;
		return projects.filter((project) => project.team_id === teamFilter);
	}, [projects, teamFilter]);

	const selectedTeamName = teamFilter ? teamsById.get(teamFilter) : undefined;

	return (
		<PageBody>
			<PageHeader
				title="Projects"
				description="Projects bind virtual API keys and configuration within a team."
				actions={
					<>
						<div className="flex items-center gap-2">
							<Label htmlFor="project-team-filter" className="sr-only">
								Team
							</Label>
							<Select
								value={teamFilter ?? "__all__"}
								onValueChange={(value) => {
									const next =
										value === "__all__" || value == null ? undefined : value;
									void navigate({
										search: (prev) => ({
											...prev,
											team: next,
										}),
										replace: true,
									});
								}}
							>
								<SelectTrigger id="project-team-filter" className="w-48">
									<SelectValue placeholder="All teams" />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="__all__">All teams</SelectItem>
									{teams.map((team) => (
										<SelectItem key={team.id} value={team.id}>
											{team.name}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
						<Button
							onClick={() => setOpen(true)}
							disabled={!activeOrg?.teams.length}
						>
							<PlusIcon />
							New project
						</Button>
					</>
				}
			/>

			<QueryGate
				isPending={isBootstrapping}
				isError={isError || projectsQuery.isError}
				error={error || projectsQuery.error}
				onRetry={refetch}
			>
				{projects.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<FolderKanbanIcon />
							</EmptyMedia>
							<EmptyTitle>No projects yet</EmptyTitle>
							<EmptyDescription>
								Create a project to issue virtual API keys for the gateway.
							</EmptyDescription>
						</EmptyHeader>
						<EmptyContent>
							<Button
								onClick={() => setOpen(true)}
								disabled={!activeOrg?.teams.length}
							>
								<PlusIcon />
								Create project
							</Button>
						</EmptyContent>
					</Empty>
				) : filteredProjects.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<FolderKanbanIcon />
							</EmptyMedia>
							<EmptyTitle>No projects for this team</EmptyTitle>
							<EmptyDescription>
								{selectedTeamName
									? `${selectedTeamName} has no projects yet.`
									: "No projects match the selected team."}
							</EmptyDescription>
						</EmptyHeader>
						<EmptyContent>
							<Button
								variant="outline"
								onClick={() =>
									void navigate({
										search: (prev) => ({ ...prev, team: undefined }),
										replace: true,
									})
								}
							>
								Clear team filter
							</Button>
						</EmptyContent>
					</Empty>
				) : (
					<div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
						{filteredProjects.map((project) => (
							<Link
								key={project.id}
								to="/app/projects/$projectId"
								params={{ projectId: project.id }}
								className="block rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							>
								<Card className="h-full transition-colors hover:bg-muted/40">
									<CardHeader>
										<CardTitle className="flex items-center gap-2">
											<FolderKanbanIcon className="size-4 text-muted-foreground" />
											{project.name}
										</CardTitle>
										<CardDescription className="font-mono text-xs">
											{project.id}
										</CardDescription>
									</CardHeader>
									<CardContent className="flex items-center justify-between gap-2">
										<Badge variant="secondary">
											{teamsById.get(project.team_id) || "Unassigned team"}
										</Badge>
										<span className="text-xs text-muted-foreground">Open</span>
									</CardContent>
								</Card>
							</Link>
						))}
					</div>
				)}
			</QueryGate>

			<CreateProjectSheet open={open} onOpenChange={setOpen} />
		</PageBody>
	);
}
