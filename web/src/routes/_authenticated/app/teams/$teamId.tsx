import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { PlusIcon, Users2Icon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { orgMembersQueryOptions } from "#/api/organization";
import {
	addTeamMemberMutationOptions,
	removeTeamMemberMutationOptions,
	teamMembersQueryOptions,
	teamQueryOptions,
} from "#/api/team";
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
import { useAuthUser } from "#/state/auth-state";
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

	const isTeamOwner = useMemo(() => {
		const me = (membersQuery.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner";
	}, [membersQuery.data, user?.id]);

	const canManage = isOrgAdmin || isTeamOwner;

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
														<Badge variant="secondary">{member.role}</Badge>
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
									addMember.isPending || !selectedUserId || candidates.length === 0
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
