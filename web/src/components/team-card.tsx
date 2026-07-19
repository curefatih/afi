import { Link } from "@tanstack/react-router";
import { ArrowRightIcon, FolderKanbanIcon } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
} from "./ui/card";

type TeamProject = {
	id: string;
	name: string;
};

type TeamCardProps = {
	id: string;
	name: string;
	description: string;
	projects: TeamProject[];
};

export default function TeamCard({
	id,
	name,
	description,
	projects,
}: TeamCardProps) {
	const previewProjects = projects.slice(0, 4);
	const remaining = projects.length - previewProjects.length;

	return (
		<Card className="flex w-full max-w-md flex-col">
			<CardHeader className="gap-2">
				<div className="flex items-start justify-between gap-3">
					<div className="min-w-0 space-y-1">
						<CardTitle className="text-lg">{name}</CardTitle>
						<CardDescription className="font-mono text-xs">
							{description}
						</CardDescription>
					</div>
					<Badge variant="secondary" className="shrink-0">
						{projects.length} {projects.length === 1 ? "project" : "projects"}
					</Badge>
				</div>
			</CardHeader>

			<CardContent className="flex flex-1 flex-col gap-2">
				{projects.length === 0 ? (
					<p className="text-sm text-muted-foreground">
						No projects yet for this team.
					</p>
				) : (
					<ul className="space-y-1.5">
						{previewProjects.map((project) => (
							<li key={project.id}>
								<Link
									to="/app/projects/$projectId"
									params={{ projectId: project.id }}
									className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
								>
									<FolderKanbanIcon className="size-3.5 shrink-0 text-muted-foreground" />
									<span className="truncate font-medium">{project.name}</span>
								</Link>
							</li>
						))}
						{remaining > 0 ? (
							<li className="px-2 text-xs text-muted-foreground">
								+{remaining} more
							</li>
						) : null}
					</ul>
				)}
			</CardContent>

			<CardFooter className="flex flex-wrap gap-2">
				<Button
					variant="outline"
					size="sm"
					nativeButton={false}
					render={<Link to="/app/teams/$teamId" params={{ teamId: id }} />}
				>
					View team
				</Button>
				<Button
					variant="ghost"
					size="sm"
					nativeButton={false}
					render={<Link to="/app/projects" search={{ team: id }} />}
				>
					View projects
					<ArrowRightIcon />
				</Button>
			</CardFooter>
		</Card>
	);
}
