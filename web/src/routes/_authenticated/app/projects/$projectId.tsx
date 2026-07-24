import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { KeyRoundIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import {
	createEnvironmentMutationOptions,
	deleteEnvironmentMutationOptions,
	environmentsQueryOptions,
} from "#/api/environment";
import { projectKeysQueryOptions } from "#/api/keys";
import { orgMembersQueryOptions } from "#/api/organization";
import { CopyableId } from "#/components/copyable-id";
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
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import {
	Sheet,
	SheetContent,
	SheetDescription,
	SheetFooter,
	SheetHeader,
	SheetTitle,
} from "#/components/ui/sheet";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { useOrgBootstrap } from "#/hooks/use-org-bootstrap";
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/projects/$projectId")(
	{
		...pageTitle("Project"),
		component: RouteComponent,
	},
);

function formatKeyPrefix(prefix: string) {
	if (!prefix) return "••••••••";
	return `${prefix}…`;
}

function RouteComponent() {
	const { projectId } = Route.useParams();
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const user = useAuthUser();
	const { isBootstrapping } = useOrgBootstrap();
	const qc = useQueryClient();
	const [open, setOpen] = useState(false);
	const [envOpen, setEnvOpen] = useState(false);
	const [envName, setEnvName] = useState("");
	const [envSlug, setEnvSlug] = useState("");

	const project = activeOrg?.projects.find((p) => p.id === projectId);
	const team = activeOrg?.teams.find((t) => t.id === project?.team_id);

	const members = useQuery(orgMembersQueryOptions(orgId));
	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const keysQuery = useQuery({
		...projectKeysQueryOptions(projectId),
	});
	const envsQuery = useQuery(environmentsQueryOptions(orgId, projectId));
	const createEnv = useMutation(createEnvironmentMutationOptions());
	const deleteEnv = useMutation(deleteEnvironmentMutationOptions());

	const envNameById = useMemo(() => {
		return new Map((envsQuery.data ?? []).map((e) => [e.id, e.name] as const));
	}, [envsQuery.data]);

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

	const invalidateEnvs = () => {
		void qc.invalidateQueries({
			queryKey: ["organizations", orgId, "projects", projectId, "environments"],
		});
	};

	return (
		<PageBody>
			<PageHeader
				title={project.name}
				description="Project service-account keys authenticate automation traffic for this project."
				actions={
					<Button onClick={() => setOpen(true)}>
						<PlusIcon />
						New service key
					</Button>
				}
			/>

			<div className="grid gap-4 md:grid-cols-3">
				<Card>
					<CardHeader>
						<CardDescription>Project ID</CardDescription>
						<CardTitle className="text-sm">
							<CopyableId value={project.id} className="text-sm" />
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
						<CardTitle>Environments</CardTitle>
						<CardDescription>
							Named stages (dev / stage / prod) for key attribution.
						</CardDescription>
					</div>
					{isOrgAdmin ? (
						<Button
							variant="outline"
							size="sm"
							onClick={() => setEnvOpen(true)}
						>
							<PlusIcon />
							Add
						</Button>
					) : null}
				</CardHeader>
				<CardContent>
					<QueryGate
						isPending={envsQuery.isPending}
						isError={envsQuery.isError}
						error={envsQuery.error}
						onRetry={() => void envsQuery.refetch()}
					>
						{(envsQuery.data?.length ?? 0) === 0 ? (
							<Empty className="border min-h-40">
								<EmptyHeader>
									<EmptyTitle>No environments</EmptyTitle>
									<EmptyDescription>
										Create environments like dev, stage, or prod, then bind
										keys to them.
									</EmptyDescription>
								</EmptyHeader>
								{isOrgAdmin ? (
									<EmptyContent>
										<Button onClick={() => setEnvOpen(true)}>
											<PlusIcon />
											Add environment
										</Button>
									</EmptyContent>
								) : null}
							</Empty>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Name</TableHead>
										<TableHead>Slug</TableHead>
										<TableHead>Created</TableHead>
										{isOrgAdmin ? <TableHead className="w-12" /> : null}
									</TableRow>
								</TableHeader>
								<TableBody>
									{envsQuery.data?.map((env) => (
										<TableRow key={env.id}>
											<TableCell className="font-medium">{env.name}</TableCell>
											<TableCell>
												<Badge variant="outline" className="font-mono">
													{env.slug}
												</Badge>
											</TableCell>
											<TableCell className="text-muted-foreground">
												{new Date(env.created_at).toLocaleString()}
											</TableCell>
											{isOrgAdmin ? (
												<TableCell>
													<Button
														variant="ghost"
														size="icon"
														disabled={deleteEnv.isPending}
														onClick={() => {
															deleteEnv.mutate(env.id, {
																onSuccess: () => {
																	invalidateEnvs();
																	toast.success("Environment deleted");
																},
																onError: (err) =>
																	toast.error(
																		err instanceof Error
																			? err.message
																			: "Delete failed",
																	),
															});
														}}
													>
														<Trash2Icon />
													</Button>
												</TableCell>
											) : null}
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</QueryGate>
				</CardContent>
			</Card>

			<Card>
				<CardHeader className="flex flex-row items-center justify-between gap-2">
					<div>
						<CardTitle>API keys</CardTitle>
						<CardDescription>
							Service-account keys scoped to this project (admin only to
							create).
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
										<TableHead>Kind</TableHead>
										<TableHead>Environment</TableHead>
										<TableHead>Key</TableHead>
										<TableHead>Created</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{keysQuery.data?.map((key) => (
										<TableRow key={key.id}>
											<TableCell className="font-medium">{key.name}</TableCell>
											<TableCell>
												<Badge variant="secondary">
													{key.kind === "personal"
														? "Personal"
														: "Service account"}
												</Badge>
											</TableCell>
											<TableCell className="text-muted-foreground">
												{key.environment_id
													? (envNameById.get(key.environment_id) ??
														key.environment_id)
													: "—"}
											</TableCell>
											<TableCell>
												<Badge variant="outline" className="font-mono">
													{formatKeyPrefix(key.key_prefix)}
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

			<Sheet open={envOpen} onOpenChange={setEnvOpen}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Add environment</SheetTitle>
						<SheetDescription>
							Slug must be unique within this project (lowercase, - / _).
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!orgId) return;
							createEnv.mutate(
								{
									orgId,
									projectId,
									name: envName,
									slug: envSlug,
								},
								{
									onSuccess: () => {
										invalidateEnvs();
										setEnvOpen(false);
										setEnvName("");
										setEnvSlug("");
										toast.success("Environment created");
									},
									onError: (err) =>
										toast.error(
											err instanceof Error ? err.message : "Create failed",
										),
								},
							);
						}}
					>
						<div className="space-y-1">
							<Label htmlFor="env-name">Name</Label>
							<Input
								id="env-name"
								value={envName}
								onChange={(e) => setEnvName(e.target.value)}
								placeholder="Production"
								required
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="env-slug">Slug</Label>
							<Input
								id="env-slug"
								value={envSlug}
								onChange={(e) => setEnvSlug(e.target.value)}
								placeholder="prod"
								required
							/>
						</div>
						<SheetFooter>
							<Button
								type="button"
								variant="outline"
								onClick={() => setEnvOpen(false)}
							>
								Cancel
							</Button>
							<Button type="submit" disabled={createEnv.isPending}>
								{createEnv.isPending ? "Creating…" : "Create"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>
		</PageBody>
	);
}
