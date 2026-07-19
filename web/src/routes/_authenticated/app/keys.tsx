import { useQueries } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { KeyRoundIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { type ApiKey, projectKeysQueryOptions } from "#/api/keys";
import { CreateKeySheet } from "#/components/create-key-sheet";
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
import { useOrgBootstrap } from "#/hooks/use-org-bootstrap";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/keys")({
	staticData: {
		getTitle: () => "API Keys",
	},
	component: RouteComponent,
});

function maskKey(key: string) {
	if (key.length <= 12) return "••••••••";
	return `${key.slice(0, 7)}…${key.slice(-4)}`;
}

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const { isBootstrapping, isError, error, refetch } = useOrgBootstrap();
	const [open, setOpen] = useState(false);
	const [projectFilter, setProjectFilter] = useState<string>("all");

	const projects = activeOrg?.projects ?? [];
	const filteredProjects =
		projectFilter === "all"
			? projects
			: projects.filter((p) => p.id === projectFilter);

	const keyQueries = useQueries({
		queries: filteredProjects.map((project) => ({
			...projectKeysQueryOptions(project.id),
		})),
	});

	const rows = useMemo(() => {
		const projectName = new Map(projects.map((p) => [p.id, p.name]));
		const items: Array<ApiKey & { projectName: string }> = [];
		keyQueries.forEach((query, index) => {
			const project = filteredProjects[index];
			if (!project || !query.data) return;
			for (const key of query.data) {
				items.push({
					...key,
					projectName: projectName.get(project.id) || project.name,
				});
			}
		});
		return items;
	}, [keyQueries, filteredProjects, projects]);

	const isPending =
		isBootstrapping ||
		keyQueries.some((q) => q.isPending && !!filteredProjects.length);
	const keysError = keyQueries.find((q) => q.isError)?.error;

	return (
		<PageBody>
			<PageHeader
				title="API Keys"
				description="Virtual API keys scoped to projects for gateway authentication."
				actions={
					<>
						<Select
							value={projectFilter}
							onValueChange={(value) => setProjectFilter(value ?? "all")}
						>
							<SelectTrigger className="w-44">
								<SelectValue placeholder="All projects" />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="all">All projects</SelectItem>
								{projects.map((project) => (
									<SelectItem key={project.id} value={project.id}>
										{project.name}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
						<Button
							onClick={() => setOpen(true)}
							disabled={projects.length === 0}
						>
							<PlusIcon />
							New key
						</Button>
					</>
				}
			/>

			<QueryGate
				isPending={isPending}
				isError={isError || !!keysError}
				error={error || keysError}
				onRetry={refetch}
			>
				{projects.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<KeyRoundIcon />
							</EmptyMedia>
							<EmptyTitle>Create a project first</EmptyTitle>
							<EmptyDescription>
								API keys belong to a project. Add a project, then issue a key.
							</EmptyDescription>
						</EmptyHeader>
						<EmptyContent>
							<Button render={<Link to="/app/projects" />}>
								Go to projects
							</Button>
						</EmptyContent>
					</Empty>
				) : rows.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<KeyRoundIcon />
							</EmptyMedia>
							<EmptyTitle>No API keys</EmptyTitle>
							<EmptyDescription>
								Issue a virtual key to authenticate playground and client
								traffic.
							</EmptyDescription>
						</EmptyHeader>
						<EmptyContent>
							<Button onClick={() => setOpen(true)}>
								<PlusIcon />
								Create API key
							</Button>
						</EmptyContent>
					</Empty>
				) : (
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Project</TableHead>
								<TableHead>Key</TableHead>
								<TableHead>Created</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{rows.map((row) => (
								<TableRow key={row.id}>
									<TableCell className="font-medium">{row.name}</TableCell>
									<TableCell>
										<Link
											to="/app/projects/$projectId"
											params={{ projectId: row.project_id }}
											className="hover:underline"
										>
											{row.projectName}
										</Link>
									</TableCell>
									<TableCell>
										<Badge variant="outline" className="font-mono">
											{maskKey(row.key)}
										</Badge>
									</TableCell>
									<TableCell className="text-muted-foreground">
										{new Date(row.created_at).toLocaleString()}
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				)}
			</QueryGate>

			<CreateKeySheet
				open={open}
				onOpenChange={setOpen}
				defaultProjectId={
					projectFilter !== "all" ? projectFilter : projects[0]?.id
				}
			/>
		</PageBody>
	);
}
