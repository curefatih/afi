import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
	ArrowRightIcon,
	FolderKanbanIcon,
	PlusIcon,
	Users2Icon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { orgMembersQueryOptions } from "#/api/organization";
import {
	addTeamMemberMutationOptions,
	removeTeamMemberMutationOptions,
	type TeamMember,
	type TeamRole,
	teamMembersQueryOptions,
	teamQueryOptions,
	updateTeamMemberRoleMutationOptions,
} from "#/api/team";
import { CopyableId } from "#/components/copyable-id";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardAction,
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
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/teams/$teamId")({
	...pageTitle("Team"),
	component: RouteComponent,
});

function RouteComponent() {
	const { teamId } = Route.useParams();
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const [addOpen, setAddOpen] = useState(false);
	const [selectedUserId, setSelectedUserId] = useState("");
	const [busyUserId, setBusyUserId] = useState<string | null>(null);

	const teamQuery = useQuery({
		...teamQueryOptions(teamId),
	});

	const membersQuery = useQuery({
		...teamMembersQueryOptions(teamId),
	});

	const orgMembers = useQuery(orgMembersQueryOptions(orgId));

	const isOrgAdmin = useMemo(() => {
		const me = (orgMembers.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [orgMembers.data, user?.id]);

	const isTeamManager = useMemo(() => {
		const me = (membersQuery.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [membersQuery.data, user?.id]);

	const canManage = isOrgAdmin || isTeamManager;

	const ownerCount = useMemo(
		() => (membersQuery.data ?? []).filter((m) => m.role === "owner").length,
		[membersQuery.data],
	);

	const candidates = useMemo(() => {
		const onTeam = new Set((membersQuery.data ?? []).map((m) => m.user_id));
		return (orgMembers.data ?? []).filter((m) => !onTeam.has(m.user_id));
	}, [orgMembers.data, membersQuery.data]);

	const addMember = useMutation({
		...addTeamMemberMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["teams", teamId, "members"] });
			toast.success("Member added");
			setAddOpen(false);
			setSelectedUserId("");
		},
		onError: (error) => {
			toast.error(error.message || "Failed to add member");
		},
	});

	const removeMember = useMutation({
		...removeTeamMemberMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["teams", teamId, "members"] });
			toast.success("Member removed");
		},
		onError: (error) => {
			toast.error(error.message || "Failed to remove member");
		},
		onSettled: () => setBusyUserId(null),
	});

	const updateRole = useMutation(updateTeamMemberRoleMutationOptions());

	async function applyRole(member: TeamMember, role: TeamRole) {
		if (role === member.role) return;
		setBusyUserId(member.user_id);
		try {
			await updateRole.mutateAsync({
				teamId,
				userId: member.user_id,
				role,
			});
			await qc.invalidateQueries({ queryKey: ["teams", teamId, "members"] });
			toast.success(`Role set to ${role}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : "Failed to update role");
		} finally {
			setBusyUserId(null);
		}
	}

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
					actions={
						canManage ? (
							<Button onClick={() => setAddOpen(true)} disabled={!orgId}>
								<PlusIcon />
								Add member
							</Button>
						) : null
					}
				/>

				<Card>
					<CardHeader>
						<CardDescription>Team ID</CardDescription>
						<CardTitle className="text-sm">
							{teamQuery.data?.id ? (
								<CopyableId value={teamQuery.data.id} className="text-sm" />
							) : null}
						</CardTitle>
					</CardHeader>
				</Card>

				<Card>
					<CardHeader>
						<CardTitle>Projects</CardTitle>
						<CardDescription>Projects owned by this team.</CardDescription>
						{projects.length > 0 ? (
							<CardAction>
								<Button
									variant="outline"
									size="sm"
									nativeButton={false}
									render={<Link to="/app/projects" search={{ team: teamId }} />}
								>
									Browse projects
									<ArrowRightIcon />
								</Button>
							</CardAction>
						) : null}
					</CardHeader>
					<CardContent>
						{projects.length === 0 ? (
							<Empty className="border min-h-40">
								<EmptyHeader>
									<EmptyMedia variant="icon">
										<FolderKanbanIcon />
									</EmptyMedia>
									<EmptyTitle>No projects</EmptyTitle>
									<EmptyDescription>
										This team does not own any projects yet.
									</EmptyDescription>
								</EmptyHeader>
								<EmptyContent>
									<Button
										variant="outline"
										size="sm"
										nativeButton={false}
										render={
											<Link to="/app/projects" search={{ team: teamId }} />
										}
									>
										Go to projects
										<ArrowRightIcon />
									</Button>
								</EmptyContent>
							</Empty>
						) : (
							<ul className="divide-y rounded-xl border">
								{projects.map((project) => (
									<li key={project.id}>
										<Link
											to="/app/projects/$projectId"
											params={{ projectId: project.id }}
											className="flex items-center gap-3 px-3 py-2.5 transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
										>
											<FolderKanbanIcon className="size-4 shrink-0 text-muted-foreground" />
											<span className="min-w-0 flex-1 truncate font-medium">
												{project.name}
											</span>
											<span className="hidden max-w-48 truncate font-mono text-xs text-muted-foreground sm:block">
												{project.id}
											</span>
											<span className="shrink-0 text-xs text-muted-foreground">
												Open
											</span>
										</Link>
									</li>
								))}
							</ul>
						)}
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<CardTitle>Members</CardTitle>
						<CardDescription>People with access to this team.</CardDescription>
					</CardHeader>
					<CardContent>
						<QueryGate
							isPending={membersQuery.isPending || orgMembers.isPending}
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
											{canManage
												? "Add organization members to this team."
												: "This team has no members listed."}
										</EmptyDescription>
									</EmptyHeader>
									{canManage ? (
										<EmptyContent>
											<Button onClick={() => setAddOpen(true)}>
												<PlusIcon />
												Add member
											</Button>
										</EmptyContent>
									) : null}
								</Empty>
							) : (
								<Table>
									<TableHeader>
										<TableRow>
											<TableHead>Name</TableHead>
											<TableHead>Email</TableHead>
											<TableHead>Role</TableHead>
											{canManage ? <TableHead className="w-28" /> : null}
										</TableRow>
									</TableHeader>
									<TableBody>
										{membersQuery.data?.map((member) => {
											const isSoleOwner =
												member.role === "owner" && ownerCount <= 1;
											return (
												<TableRow key={member.user_id}>
													<TableCell className="font-medium">
														{member.name}
													</TableCell>
													<TableCell>{member.email}</TableCell>
													<TableCell>
														{canManage ? (
															<Select
																value={member.role}
																disabled={
																	busyUserId === member.user_id ||
																	(isSoleOwner && member.role === "owner")
																}
																onValueChange={(v) => {
																	const role = (v ?? member.role) as TeamRole;
																	void applyRole(member, role);
																}}
															>
																<SelectTrigger className="w-36">
																	<SelectValue />
																</SelectTrigger>
																<SelectContent>
																	<SelectItem value="member">
																		member
																	</SelectItem>
																	<SelectItem value="admin">admin</SelectItem>
																	<SelectItem
																		value="owner"
																		disabled={isSoleOwner}
																	>
																		owner
																	</SelectItem>
																</SelectContent>
															</Select>
														) : (
															<Badge variant="secondary">
																{member.role}
															</Badge>
														)}
													</TableCell>
													{canManage ? (
														<TableCell>
															<Button
																variant="ghost"
																size="sm"
																disabled={
																	isSoleOwner || busyUserId === member.user_id
																}
																onClick={() => {
																	setBusyUserId(member.user_id);
																	removeMember.mutate({
																		teamId,
																		userId: member.user_id,
																	});
																}}
															>
																Remove
															</Button>
														</TableCell>
													) : null}
												</TableRow>
											);
										})}
									</TableBody>
								</Table>
							)}
						</QueryGate>
					</CardContent>
				</Card>
			</QueryGate>

			<Sheet open={addOpen} onOpenChange={setAddOpen}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Add member</SheetTitle>
						<SheetDescription>
							Assign an existing organization member to this team.
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!selectedUserId) return;
							addMember.mutate({ teamId, user_id: selectedUserId });
						}}
					>
						<div className="space-y-1">
							<Label>Member</Label>
							<Select
								value={selectedUserId}
								onValueChange={(value) => setSelectedUserId(value ?? "")}
							>
								<SelectTrigger className="w-full">
									<SelectValue placeholder="Select a member" />
								</SelectTrigger>
								<SelectContent>
									{candidates.map((m) => (
										<SelectItem key={m.user_id} value={m.user_id}>
											{m.name} ({m.email})
										</SelectItem>
									))}
								</SelectContent>
							</Select>
							{candidates.length === 0 ? (
								<p className="text-muted-foreground text-xs">
									Every organization member is already on this team.
								</p>
							) : null}
						</div>
						<SheetFooter>
							<Button
								type="button"
								variant="outline"
								onClick={() => setAddOpen(false)}
							>
								Cancel
							</Button>
							<Button
								type="submit"
								disabled={
									addMember.isPending ||
									!selectedUserId ||
									candidates.length === 0
								}
							>
								{addMember.isPending ? "Adding…" : "Add member"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>
		</PageBody>
	);
}
