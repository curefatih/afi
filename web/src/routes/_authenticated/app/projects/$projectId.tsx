import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { KeyRoundIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { projectKeysQueryOptions } from "#/api/keys";
import { CreateKeySheet } from "#/components/create-key-sheet";
import { PageBody, PageHeader } from "#/components/page-header";
import { PageSkeleton, QueryError, QueryGate } from "#/components/query-state";
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

export const Route = createFileRoute("/_authenticated/app/projects/$projectId")(
	{
		staticData: {
			getTitle: () => "Project",
		},
		component: RouteComponent,
	},
);

function maskKey(key: string) {
	if (key.length <= 12) return "••••••••";
	return `${key.slice(0, 7)}…${key.slice(-4)}`;
}

function RouteComponent() {
	const { projectId } = Route.useParams();
	const activeOrg = useActiveOrg();
	const { isBootstrapping } = useOrgBootstrap();
	const [open, setOpen] = useState(false);

	const project = activeOrg?.projects.find((p) => p.id === projectId);
	const team = activeOrg?.teams.find((t) => t.id === project?.team_id);

	const keysQuery = useQuery({
		...projectKeysQueryOptions(projectId),
	});

	if (isBootstrapping && !project) {
		return (
			<PageBody>
				<PageSkeleton rows={3} />
			</PageBody>
		);
	}

	if (!project) {
		return (
			<PageBody>
				<QueryError message="Project not found in the active organization." />
			</PageBody>
		);
	}

	return (
		<PageBody>
			<PageHeader
				title={project.name}
				description="Virtual API keys and project metadata for gateway access."
				actions={
					<Button onClick={() => setOpen(true)}>
						<PlusIcon />
						New API key
					</Button>
				}
			/>

			<div className="grid gap-4 md:grid-cols-3">
				<Card>
					<CardHeader>
						<CardDescription>Project ID</CardDescription>
						<CardTitle className="font-mono text-sm break-all">
							{project.id}
						</CardTitle>
					</CardHeader>
				</Card>
				<Card>
					<CardHeader>
						<CardDescription>Team</CardDescription>
						<CardTitle className="text-base">
							{team ? (
								<Link
									to="/app/teams/$teamId"
									params={{ teamId: team.id }}
									className="hover:underline"
								>
									{team.name}
								</Link>
							) : (
								"—"
							)}
						</CardTitle>
					</CardHeader>
				</Card>
				<Card>
					<CardHeader>
						<CardDescription>API keys</CardDescription>
						<CardTitle className="text-base">
							{keysQuery.data?.length ?? "—"}
						</CardTitle>
					</CardHeader>
				</Card>
			</div>

			<Card>
				<CardHeader className="flex flex-row items-center justify-between gap-2">
					<div>
						<CardTitle>API keys</CardTitle>
						<CardDescription>
							Authenticate inference requests against this project.
						</CardDescription>
					</div>
					<Button variant="outline" size="sm" onClick={() => setOpen(true)}>
						<PlusIcon />
						Create
					</Button>
				</CardHeader>
				<CardContent>
					<QueryGate
						isPending={keysQuery.isPending}
						isError={keysQuery.isError}
						error={keysQuery.error}
						onRetry={() => void keysQuery.refetch()}
					>
						{(keysQuery.data?.length ?? 0) === 0 ? (
							<Empty className="border min-h-48">
								<EmptyHeader>
									<EmptyMedia variant="icon">
										<KeyRoundIcon />
									</EmptyMedia>
									<EmptyTitle>No keys yet</EmptyTitle>
									<EmptyDescription>
										Create a virtual API key to call the gateway.
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
										<TableHead>Key</TableHead>
										<TableHead>Created</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{keysQuery.data?.map((key) => (
										<TableRow key={key.id}>
											<TableCell className="font-medium">{key.name}</TableCell>
											<TableCell>
												<Badge variant="outline" className="font-mono">
													{maskKey(key.key)}
												</Badge>
											</TableCell>
											<TableCell className="text-muted-foreground">
												{new Date(key.created_at).toLocaleString()}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</QueryGate>
				</CardContent>
			</Card>

			<CreateKeySheet
				open={open}
				onOpenChange={setOpen}
				defaultProjectId={project.id}
			/>
		</PageBody>
	);
}
