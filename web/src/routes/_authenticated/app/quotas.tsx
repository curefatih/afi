import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { GaugeIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { orgKeysQueryOptions } from "#/api/keys";
import { orgMembersQueryOptions } from "#/api/organization";
import {
	createQuotaMutationOptions,
	deleteQuotaMutationOptions,
	type Quota,
	quotasQueryOptions,
	updateQuotaMutationOptions,
} from "#/api/quota";
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

export const Route = createFileRoute("/_authenticated/app/quotas")({
	...pageTitle("Quotas"),
	component: RouteComponent,
});

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const quotas = useQuery(quotasQueryOptions(orgId));
	const members = useQuery(orgMembersQueryOptions(orgId));
	const keys = useQuery(orgKeysQueryOptions(orgId));
	const [createOpen, setCreateOpen] = useState(false);
	const [edit, setEdit] = useState<Quota | null>(null);

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const labels = useMemo(() => {
		const projects = new Map(
			(org?.projects ?? []).map((p) => [p.id, p.name] as const),
		);
		const users = new Map(
			(members.data ?? []).map((m) => [m.user_id, m.email] as const),
		);
		const keyNames = new Map(
			(keys.data ?? []).map((k) => [k.id, k.name] as const),
		);
		return (scopeType: string, scopeId: string) => {
			switch (scopeType) {
				case "organization":
					return org?.name ?? scopeId;
				case "project":
					return projects.get(scopeId) ?? scopeId;
				case "user":
					return users.get(scopeId) ?? scopeId;
				case "api_key":
					return keyNames.get(scopeId) ?? scopeId;
				default:
					return scopeId;
			}
		};
	}, [org, members.data, keys.data]);

	const create = useMutation({
		...createQuotaMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "quotas"],
			});
			toast.success("Quota created");
			setCreateOpen(false);
		},
	});
	const update = useMutation({
		...updateQuotaMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "quotas"],
			});
			toast.success("Quota updated");
			setEdit(null);
		},
	});
	const del = useMutation({
		...deleteQuotaMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "quotas"],
			});
			toast.success("Quota deleted");
		},
	});

	const [scopeType, setScopeType] = useState("organization");
	const [scopeID, setScopeID] = useState(orgId);
	const [metric, setMetric] = useState("requests");
	const [window, setWindow] = useState("total");
	const [limitValue, setLimitValue] = useState("100");
	const [error, setError] = useState<string | null>(null);

	const [editLimit, setEditLimit] = useState("100");
	const [editError, setEditError] = useState<string | null>(null);

	const openEdit = (q: Quota) => {
		setEdit(q);
		setEditLimit(String(q.limit_value));
		setEditError(null);
	};

	const effectiveScopeID = scopeType === "organization" ? orgId : scopeID;
	const quotaList = quotas.data ?? [];

	return (
		<PageBody>
			<PageHeader
				title="Quotas"
				description="Request and token limits. Windows: total (Postgres lifetime) or minute/hour/day (Redis rate limits). Most specific wins per window: api key → user → project → organization."
				actions={
					isOrgAdmin ? (
						<Button onClick={() => setCreateOpen(true)} disabled={!orgId}>
							<PlusIcon />
							Add quota
						</Button>
					) : null
				}
			/>
			<QueryGate
				isPending={quotas.isPending || members.isPending}
				isError={quotas.isError}
				error={quotas.error}
				onRetry={() => quotas.refetch()}
			>
				{quotaList.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<GaugeIcon />
							</EmptyMedia>
							<EmptyTitle>No quotas</EmptyTitle>
							<EmptyDescription>
								Traffic is unlimited until you add a quota.
								{!isOrgAdmin
									? " Only organization owners and admins can create quotas."
									: ""}
							</EmptyDescription>
						</EmptyHeader>
						{isOrgAdmin ? (
							<EmptyContent>
								<Button onClick={() => setCreateOpen(true)} disabled={!orgId}>
									<PlusIcon />
									Add quota
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<>
						{!isOrgAdmin ? (
							<p className="text-muted-foreground text-sm">
								Only organization owners and admins can create or edit quotas.
							</p>
						) : null}
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Metric</TableHead>
									<TableHead>Limit</TableHead>
									<TableHead>Scope</TableHead>
									<TableHead>Scope ID</TableHead>
									{isOrgAdmin ? <TableHead className="w-40" /> : null}
								</TableRow>
							</TableHeader>
							<TableBody>
								{quotaList.map((q) => (
									<TableRow key={q.id}>
										<TableCell className="font-medium">{q.metric}</TableCell>
										<TableCell>
											{q.limit_value} ({q.window})
										</TableCell>
										<TableCell>
											{q.scope_type}: {labels(q.scope_type, q.scope_id)}
										</TableCell>
										<TableCell className="text-muted-foreground font-mono text-xs">
											{q.scope_id}
										</TableCell>
										{isOrgAdmin ? (
											<TableCell className="space-x-2">
												<Button
													variant="outline"
													size="sm"
													onClick={() => openEdit(q)}
												>
													Edit
												</Button>
												<Button
													variant="outline"
													size="sm"
													disabled={del.isPending}
													onClick={() => del.mutate(q.id)}
												>
													Delete
												</Button>
											</TableCell>
										) : null}
									</TableRow>
								))}
							</TableBody>
						</Table>
					</>
				)}
			</QueryGate>

			<Sheet open={createOpen} onOpenChange={setCreateOpen}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Add quota</SheetTitle>
						<SheetDescription>
							Publishes a new gateway snapshot with this limit.
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
									scope_type: scopeType,
									scope_id: effectiveScopeID,
									metric,
									limit_value: Number(limitValue),
									window,
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
							<Label>Scope</Label>
							<Select
								value={scopeType}
								onValueChange={(v) => {
									const next = v ?? "organization";
									setScopeType(next);
									if (next === "organization") setScopeID(orgId);
									else if (next === "project")
										setScopeID(org?.projects[0]?.id ?? "");
									else if (next === "user")
										setScopeID(members.data?.[0]?.user_id ?? "");
									else if (next === "api_key")
										setScopeID(keys.data?.[0]?.id ?? "");
								}}
							>
								<SelectTrigger className="w-full">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="organization">Organization</SelectItem>
									<SelectItem value="project">Project</SelectItem>
									<SelectItem value="user">User</SelectItem>
									<SelectItem value="api_key">API key</SelectItem>
								</SelectContent>
							</Select>
						</div>

						{scopeType === "project" ? (
							<div className="space-y-1">
								<Label>Project</Label>
								<Select
									value={scopeID}
									onValueChange={(v) => setScopeID(v ?? "")}
								>
									<SelectTrigger className="w-full">
										<SelectValue placeholder="Select project" />
									</SelectTrigger>
									<SelectContent>
										{(org?.projects ?? []).map((p) => (
											<SelectItem key={p.id} value={p.id}>
												{p.name}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
						) : null}

						{scopeType === "user" ? (
							<div className="space-y-1">
								<Label>Member</Label>
								<Select
									value={scopeID}
									onValueChange={(v) => setScopeID(v ?? "")}
								>
									<SelectTrigger className="w-full">
										<SelectValue placeholder="Select member" />
									</SelectTrigger>
									<SelectContent>
										{(members.data ?? []).map((m) => (
											<SelectItem key={m.user_id} value={m.user_id}>
												{m.email}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
						) : null}

						{scopeType === "api_key" ? (
							<div className="space-y-1">
								<Label>API key</Label>
								<Select
									value={scopeID}
									onValueChange={(v) => setScopeID(v ?? "")}
								>
									<SelectTrigger className="w-full">
										<SelectValue placeholder="Select key" />
									</SelectTrigger>
									<SelectContent>
										{(keys.data ?? []).map((k) => (
											<SelectItem key={k.id} value={k.id}>
												{k.name} ({k.kind})
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
						) : null}

						<div className="space-y-1">
							<Label>Metric</Label>
							<Select
								value={metric}
								onValueChange={(v) => setMetric(v ?? "requests")}
							>
								<SelectTrigger className="w-full">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="requests">requests</SelectItem>
									<SelectItem value="tokens">tokens</SelectItem>
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1">
							<Label>Window</Label>
							<Select
								value={window}
								onValueChange={(v) => setWindow(v ?? "total")}
							>
								<SelectTrigger className="w-full">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="total">total (Postgres)</SelectItem>
									<SelectItem value="minute">minute (Redis)</SelectItem>
									<SelectItem value="hour">hour (Redis)</SelectItem>
									<SelectItem value="day">day (Redis)</SelectItem>
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1">
							<Label htmlFor="q-limit">Limit</Label>
							<Input
								id="q-limit"
								type="number"
								min={0}
								value={limitValue}
								onChange={(e) => setLimitValue(e.target.value)}
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
							<Button
								type="submit"
								disabled={create.isPending || !orgId || !effectiveScopeID}
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
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Edit quota</SheetTitle>
						<SheetDescription>
							Scope, metric, and window are fixed after create. Update the
							limit value.
						</SheetDescription>
					</SheetHeader>
					{edit ? (
						<form
							className="flex flex-1 flex-col gap-4 px-4"
							onSubmit={(e) => {
								e.preventDefault();
								setEditError(null);
								update.mutate(
									{
										quotaId: edit.id,
										limit_value: Number(editLimit),
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
								<Label>Scope</Label>
								<Input
									readOnly
									value={`${edit.scope_type}: ${labels(edit.scope_type, edit.scope_id)}`}
									className="bg-muted"
								/>
							</div>
							<div className="space-y-1">
								<Label>Metric</Label>
								<Input readOnly value={edit.metric} className="bg-muted" />
							</div>
							<div className="space-y-1">
								<Label>Window</Label>
								<Input readOnly value={edit.window} className="bg-muted" />
							</div>
							<div className="space-y-1">
								<Label htmlFor="edit-q-limit">Limit</Label>
								<Input
									id="edit-q-limit"
									type="number"
									min={0}
									value={editLimit}
									onChange={(e) => setEditLimit(e.target.value)}
									required
								/>
							</div>
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
								<Button type="submit" disabled={update.isPending}>
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
