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
	type RequestPolicy,
	reorderPoliciesMutationOptions,
	updatePolicyMutationOptions,
} from "#/api/policies";
import { PageBody, PageHeader } from "#/components/page-header";
import { CelExpressionEditor } from "#/components/policies/cel-expression-editor";
import { SortablePolicyTable } from "#/components/policies/sortable-policy-table";
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
import { Switch } from "#/components/ui/switch";
import { CEL_EXAMPLES, CEL_VARIABLES } from "#/lib/cel-policy-catalog";
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/policies")({
	...pageTitle("Policies"),
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
	const [edit, setEdit] = useState<RequestPolicy | null>(null);

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
			setName("");
			setExpression('request.model != "blocked-model"');
			setPriority("100");
			setError(null);
		},
	});
	const update = useMutation({
		...updatePolicyMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "policies"],
			});
			toast.success("Policy updated");
			setEdit(null);
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
	const reorder = useMutation({
		...reorderPoliciesMutationOptions(),
		onMutate: async (vars) => {
			const queryKey = ["organizations", orgId, "policies"] as const;
			await qc.cancelQueries({ queryKey });
			const previous = qc.getQueryData<RequestPolicy[]>(queryKey);
			if (previous) {
				const byId = new Map(previous.map((p) => [p.id, p]));
				qc.setQueryData<RequestPolicy[]>(
					queryKey,
					vars.policies.flatMap((p) => {
						const cur = byId.get(p.id);
						return cur ? [{ ...cur, priority: p.priority }] : [];
					}),
				);
			}
			return { previous };
		},
		onError: (err, _vars, ctx) => {
			if (ctx?.previous) {
				qc.setQueryData(["organizations", orgId, "policies"], ctx.previous);
			}
			toast.error(
				err instanceof Error ? err.message : "Failed to reorder policies",
			);
		},
		onSuccess: () => {
			toast.success("Policy order updated");
		},
		onSettled: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "policies"],
			});
		},
	});

	const [name, setName] = useState("");
	const [expression, setExpression] = useState(
		'request.model != "blocked-model"',
	);
	const [priority, setPriority] = useState("100");
	const [error, setError] = useState<string | null>(null);

	const [editName, setEditName] = useState("");
	const [editExpression, setEditExpression] = useState("");
	const [editPriority, setEditPriority] = useState("100");
	const [editEnabled, setEditEnabled] = useState(true);
	const [editError, setEditError] = useState<string | null>(null);

	const openEdit = (p: RequestPolicy) => {
		setEdit(p);
		setEditName(p.name);
		setEditExpression(p.expression);
		setEditPriority(String(p.priority));
		setEditEnabled(p.enabled);
		setEditError(null);
	};

	const list = useMemo(() => {
		const rows = [...(policies.data ?? [])];
		rows.sort((a, b) => {
			if (a.priority !== b.priority) return b.priority - a.priority;
			return a.name.localeCompare(b.name);
		});
		return rows;
	}, [policies.data]);

	const handleReorder = (next: RequestPolicy[]) => {
		const sameOrder =
			next.length === list.length &&
			next.every(
				(p, i) => p.id === list[i]?.id && p.priority === list[i]?.priority,
			);
		if (sameOrder) return;
		reorder.mutate({
			policies: next.map((p) => ({ id: p.id, priority: p.priority })),
			previous: list.map((p) => ({ id: p.id, priority: p.priority })),
		});
	};
	return (
		<PageBody>
			<PageHeader
				title="Policies"
				description="Allow-rules for gateway traffic. Each enabled CEL expression must return true or the request is denied with HTTP 403."
				actions={
					isOrgAdmin ? (
						<Button onClick={() => setCreateOpen(true)} disabled={!orgId}>
							<PlusIcon />
							Add policy
						</Button>
					) : null
				}
			/>

			<div className="rounded-lg border bg-muted/20 p-4 text-sm space-y-3">
				<div>
					<p className="font-medium">Quick start</p>
					<p className="text-muted-foreground text-xs mt-1 leading-relaxed">
						Policies are boolean CEL expressions evaluated after auth. Use{" "}
						<code className="text-foreground">request.*</code> for the call and{" "}
						<code className="text-foreground">key.*</code> for the virtual API
						key. All enabled policies in the org must pass.
					</p>
				</div>
				<div className="flex flex-wrap gap-1.5">
					{CEL_VARIABLES.filter((v) => v.type === "field").map((v) => (
						<code
							key={v.label}
							className="rounded-md border bg-background px-1.5 py-0.5 font-mono text-[11px]"
							title={v.detail}
						>
							{v.label}
						</code>
					))}
				</div>
				<div className="grid gap-2 sm:grid-cols-2">
					{CEL_EXAMPLES.slice(0, 4).map((ex) => (
						<div
							key={ex.title}
							className="rounded-md border bg-background/80 px-3 py-2"
						>
							<p className="text-xs font-medium">{ex.title}</p>
							<code className="mt-1 block font-mono text-[11px] text-muted-foreground truncate">
								{ex.expression}
							</code>
						</div>
					))}
				</div>
			</div>

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
							<EmptyTitle>No policies yet</EmptyTitle>
							<EmptyDescription>
								Traffic is allowed until you add a rule. Start from an example
								in the editor — block a model, disallow streaming, or lock keys
								to personal only.
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
					<>
						{!isOrgAdmin ? (
							<p className="text-muted-foreground text-sm">
								Only organization owners and admins can create or edit policies.
							</p>
						) : null}
						<SortablePolicyTable
							policies={list}
							canEdit={isOrgAdmin}
							disabled={reorder.isPending}
							onReorder={handleReorder}
							onEdit={openEdit}
							onDelete={(id) => del.mutate(id)}
							deletePending={del.isPending}
						/>
					</>
				)}
			</QueryGate>

			<Sheet open={createOpen} onOpenChange={setCreateOpen}>
				<SheetContent className="w-full overflow-y-auto sm:max-w-2xl data-[side=right]:sm:max-w-2xl data-[side=left]:sm:max-w-2xl">
					<SheetHeader>
						<SheetTitle>Add CEL policy</SheetTitle>
						<SheetDescription>
							Must evaluate to bool. Denial returns HTTP 403 policy_violation.
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4 pb-4"
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
								placeholder="block-risky-model"
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
							<p className="text-[11px] text-muted-foreground">
								Higher priority runs first. You can also drag rows in the table
								to set order.
							</p>
						</div>
						<CelExpressionEditor
							id="pol-expr"
							value={expression}
							onChange={setExpression}
						/>
						{error ? <p className="text-destructive text-xs">{error}</p> : null}
						<SheetFooter>
							<Button
								type="button"
								variant="outline"
								onClick={() => setCreateOpen(false)}
							>
								Cancel
							</Button>
							<Button
								type="submit"
								disabled={create.isPending || !orgId || !name.trim()}
							>
								{create.isPending ? "Creating…" : "Create & publish"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>

			<Sheet
				open={!!edit}
				onOpenChange={(o) => {
					if (!o) setEdit(null);
				}}
			>
				<SheetContent className="w-full overflow-y-auto sm:max-w-2xl data-[side=right]:sm:max-w-2xl data-[side=left]:sm:max-w-2xl">
					<SheetHeader>
						<SheetTitle>Edit CEL policy</SheetTitle>
						<SheetDescription>
							Update name, priority, expression, or enable/disable. Denial
							returns HTTP 403 policy_violation.
						</SheetDescription>
					</SheetHeader>
					{edit ? (
						<form
							className="flex flex-1 flex-col gap-4 px-4 pb-4"
							onSubmit={(e) => {
								e.preventDefault();
								setEditError(null);
								update.mutate(
									{
										policyId: edit.id,
										name: editName,
										expression: editExpression,
										priority:
											editPriority.trim() === "" ||
											Number.isNaN(Number(editPriority))
												? 100
												: Number(editPriority),
										enabled: editEnabled,
									},
									{
										onError: (err) =>
											setEditError(
												err instanceof Error ? err.message : "Update failed",
											),
									},
								);
							}}
						>
							<div className="space-y-1">
								<Label htmlFor="edit-pol-name">Name</Label>
								<Input
									id="edit-pol-name"
									value={editName}
									onChange={(e) => setEditName(e.target.value)}
									required
								/>
							</div>
							<div className="space-y-1">
								<Label htmlFor="edit-pol-priority">Priority</Label>
								<Input
									id="edit-pol-priority"
									type="number"
									value={editPriority}
									onChange={(e) => setEditPriority(e.target.value)}
								/>
								<p className="text-[11px] text-muted-foreground">
									Higher priority runs first. You can also drag rows in the
									table to set order.
								</p>
							</div>
							<div className="flex items-center justify-between gap-2">
								<div>
									<Label htmlFor="edit-pol-enabled">Enabled</Label>
									<p className="text-[11px] text-muted-foreground">
										Disabled policies are skipped at evaluation time.
									</p>
								</div>
								<Switch
									id="edit-pol-enabled"
									checked={editEnabled}
									onCheckedChange={setEditEnabled}
								/>
							</div>
							<CelExpressionEditor
								id="edit-pol-expr"
								value={editExpression}
								onChange={setEditExpression}
							/>
							{editError ? (
								<p className="text-destructive text-xs">{editError}</p>
							) : null}
							<SheetFooter>
								<Button
									type="button"
									variant="outline"
									onClick={() => setEdit(null)}
								>
									Cancel
								</Button>
								<Button
									type="submit"
									disabled={update.isPending || !editName.trim()}
								>
									{update.isPending ? "Saving…" : "Save & publish"}
								</Button>
							</SheetFooter>
						</form>
					) : null}
				</SheetContent>
			</Sheet>
		</PageBody>
	);
}
