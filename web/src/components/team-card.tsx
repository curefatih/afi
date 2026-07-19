import { Link } from "@tanstack/react-router";
import { FolderKanbanIcon, Users2Icon } from "lucide-react";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "./ui/card";

const PREVIEW_LIMIT = 2;

type TeamProject = {
	id: string;
	name: string;
};

type TeamCardProps = {
	id: string;
	name: string;
	projects: TeamProject[];
};

export default function TeamCard({ id, name, projects }: TeamCardProps) {
	const hasMore = projects.length > PREVIEW_LIMIT;
	const previewProjects = hasMore ? projects.slice(0, PREVIEW_LIMIT) : projects;
	const projectLabel =
		projects.length === 1 ? "1 project" : `${projects.length} projects`;

	return (
		<Card className="h-full transition-colors hover:bg-muted/40">
			<CardHeader>
				<Link
					to="/app/teams/$teamId"
					params={{ teamId: id }}
					className="group/team block space-y-1 rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
				>
					<CardTitle className="flex items-center gap-2">
						<Users2Icon className="size-4 text-muted-foreground" />
						<span className="truncate group-hover/team:underline">{name}</span>
					</CardTitle>
					<CardDescription>{projectLabel}</CardDescription>
				</Link>
			</CardHeader>

			<CardContent className="flex flex-1 flex-col">
				{projects.length === 0 ? (
					<div className="flex flex-1 items-center rounded-lg border border-dashed px-3 py-4 text-sm text-muted-foreground">
						No projects yet
					</div>
				) : (
					<div className="flex flex-1 flex-col gap-1 rounded-lg bg-muted/40 p-1">
						{previewProjects.map((project) => (
							<Link
								key={project.id}
								to="/app/projects/$projectId"
								params={{ projectId: project.id }}
								className="flex items-center gap-2 rounded-md px-2.5 py-2 text-sm transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							>
								<FolderKanbanIcon className="size-3.5 shrink-0 text-muted-foreground" />
								<span className="truncate">{project.name}</span>
							</Link>
						))}
						{hasMore ? (
							<Link
								to="/app/projects"
								search={{ team: id }}
								className="rounded-md px-2.5 py-2 text-sm text-muted-foreground transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							>
								More…
							</Link>
						) : null}
					</div>
				)}
			</CardContent>
		</Card>
	);
}
