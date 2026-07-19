import { createFileRoute, Link } from "@tanstack/react-router";
import { FolderKanbanIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
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
import { useOrgBootstrap } from "#/hooks/use-org-bootstrap";
import { pageTitle } from "#/lib/page-meta";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/projects/")({
	...pageTitle("Projects"),
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const { isBootstrapping, isError, error, refetch, projectsQuery } =
		useOrgBootstrap();
	const [open, setOpen] = useState(false);

	const projects = activeOrg?.projects ?? [];
	const teamsById = new Map(
		(activeOrg?.teams ?? []).map((team) => [team.id, team.name]),
	);

	return (
		<PageBody>
			<PageHeader
				title="Projects"
				description="Projects bind virtual API keys and configuration within a team."
				actions={
					<Button
						onClick={() => setOpen(true)}
						disabled={!activeOrg?.teams.length}
					>
						<PlusIcon />
						New project
					</Button>
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
				) : (
					<div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
						{projects.map((project) => (
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
