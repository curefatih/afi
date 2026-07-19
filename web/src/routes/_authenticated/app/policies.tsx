import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { PlusIcon, ShieldCheckIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { orgMembersQueryOptions } from "#/api/organization";
import {
	createPolicyMutationOptions,
	deletePolicyMutationOptions,
	policiesQueryOptions,
} from "#/api/policies";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Button } from "#/components/ui/button";
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
import { Textarea } from "#/components/ui/textarea";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/policies")({
	staticData: {
		getTitle: () => "Policies",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const policies = useQuery(policiesQueryOptions(orgId));
	const members = useQuery(orgMembersQueryOptions(orgId));
	const [createOpen, setCreateOpen] = useState(false);

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const create = useMutation({
		...createPolicyMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "policies"],
			});
			toast.success("Policy created");
			setCreateOpen(false);
		},
	});
	const del = useMutation({
		...deletePolicyMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "policies"],
			});
			toast.success("Policy deleted");
		},
	});

	const [name, setName] = useState("allow-echo");
	const [expression, setExpression] = useState(
		'request.model != "blocked-model"',
	);
	const [priority, setPriority] = useState("100");
	const [error, setError] = useState<string | null>(null);
	const list = policies.data ?? [];

	return (
		<PageBody>
			<PageHeader
				title="Policies"
				description="CEL allow-expressions evaluated on every gateway request. All enabled org policies must return true. Variables: request.model, request.path, request.stream, key.id, key.organization_id, key.project_id, key.kind, key.owner_user_id."
				actions={
					isOrgAdmin ? (
						<Button onClick={() => setCreateOpen(true)} disabled={!orgId}>
							<PlusIcon />
							Add policy
						</Button>
					) : null
				}
			/>
			<QueryGate
				isPending={policies.isPending || members.isPending}
				isError={policies.isError}
				error={policies.error}
				onRetry={() => policies.refetch()}
			>
				{list.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<ShieldCheckIcon />
							</EmptyMedia>
							<EmptyTitle>No policies</EmptyTitle>
							<EmptyDescription>
								Requests are allowed until you add a CEL policy.
								{!isOrgAdmin
									? " Only organization owners and admins can create policies."
									: ""}
							</EmptyDescription>
						</EmptyHeader>
						{isOrgAdmin ? (
							<EmptyContent>
								<Button onClick={() => setCreateOpen(true)} disabled={!orgId}>
									<PlusIcon />
									Add policy
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Priority</TableHead>
								<TableHead>Enabled</TableHead>
								<TableHead>Expression</TableHead>
								{isOrgAdmin ? <TableHead className="w-24" /> : null}
							</TableRow>
						</TableHeader>
						<TableBody>
							{list.map((p) => (
								<TableRow key={p.id}>
									<TableCell className="font-medium">{p.name}</TableCell>
									<TableCell>{p.priority}</TableCell>
									<TableCell>{p.enabled ? "yes" : "no"}</TableCell>
									<TableCell className="font-mono text-xs max-w-md truncate">
										{p.expression}
									</TableCell>
									{isOrgAdmin ? (
										<TableCell>
											<Button
												variant="outline"
												size="sm"
												disabled={del.isPending}
												onClick={() => del.mutate(p.id)}
											>
												Delete
											</Button>
										</TableCell>
									) : null}
								</TableRow>
							))}
						</TableBody>
					</Table>
				)}
			</QueryGate>

			<Sheet open={createOpen} onOpenChange={setCreateOpen}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Add CEL policy</SheetTitle>
						<SheetDescription>
							Expression must evaluate to bool. Denial returns HTTP 403.
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!orgId) return;
							setError(null);
							create.mutate(
								{
									orgId,
									name,
									expression,
									priority: Number(priority) || 100,
								},
								{
									onError: (err) =>
										setError(
											err instanceof Error ? err.message : "Create failed",
										),
								},
							);
						}}
					>
						<div className="space-y-1">
							<Label htmlFor="pol-name">Name</Label>
							<Input
								id="pol-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="pol-priority">Priority</Label>
							<Input
								id="pol-priority"
								type="number"
								value={priority}
								onChange={(e) => setPriority(e.target.value)}
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="pol-expr">Expression</Label>
							<Textarea
								id="pol-expr"
								value={expression}
								onChange={(e) => setExpression(e.target.value)}
								rows={5}
								className="font-mono text-xs"
								required
							/>
						</div>
						{error ? <p className="text-destructive text-xs">{error}</p> : null}
						<SheetFooter>
							<Button
								type="button"
								variant="outline"
								onClick={() => setCreateOpen(false)}
							>
								Cancel
							</Button>
							<Button type="submit" disabled={create.isPending || !orgId}>
								{create.isPending ? "Creating…" : "Create & publish"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>
		</PageBody>
	);
}
