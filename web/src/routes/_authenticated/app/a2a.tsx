import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { BotIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import {
	type A2AAgent,
	a2aAgentsQueryOptions,
	createA2AAgentMutationOptions,
	deleteA2AAgentMutationOptions,
	type ProtocolProbeResult,
	testA2AAgentMutationOptions,
	updateA2AAgentMutationOptions,
} from "#/api/a2a";
import { orgMembersQueryOptions } from "#/api/organization";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Checkbox } from "#/components/ui/checkbox";
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
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/a2a")({
	...pageTitle("A2A"),
	component: RouteComponent,
});

function blankForm() {
	return {
		alias: "",
		name: "",
		upstream_url: "",
		card_url: "",
		card_cache: "",
		api_key_env: "",
		auth_scheme: "",
		enabled: true,
	};
}

function parseCardCache(raw: string): unknown | undefined {
	const t = raw.trim();
	if (!t) return undefined;
	return JSON.parse(t) as unknown;
}

function formatCardCache(cache: unknown): string {
	if (cache == null || cache === "") return "";
	try {
		return JSON.stringify(cache, null, 2);
	} catch {
		return String(cache);
	}
}

function probeToast(res: ProtocolProbeResult) {
	const latency = `${res.latency_ms}ms`;
	if (res.ok) {
		toast.success(`Connected (HTTP ${res.status_code ?? "—"}, ${latency})`);
		return;
	}
	toast.error(
		res.error ||
			`Connection failed (HTTP ${res.status_code ?? "—"}, ${latency})`,
	);
}

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const agents = useQuery(a2aAgentsQueryOptions(orgId));
	const members = useQuery(orgMembersQueryOptions(orgId));
	const [createOpen, setCreateOpen] = useState(false);
	const [edit, setEdit] = useState<A2AAgent | null>(null);
	const [form, setForm] = useState(blankForm);
	const [editForm, setEditForm] = useState(blankForm);
	const [error, setError] = useState<string | null>(null);
	const [editError, setEditError] = useState<string | null>(null);

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const invalidate = () =>
		void qc.invalidateQueries({
			queryKey: ["organizations", orgId, "a2a-agents"],
		});

	const create = useMutation({
		...createA2AAgentMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("A2A agent created");
			setCreateOpen(false);
			setForm(blankForm());
		},
	});
	const update = useMutation({
		...updateA2AAgentMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("A2A agent updated");
			setEdit(null);
		},
	});
	const del = useMutation({
		...deleteA2AAgentMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("A2A agent deleted");
		},
	});
	const testConn = useMutation({
		...testA2AAgentMutationOptions(),
		onSuccess: (res) => probeToast(res),
		onError: (e) => toast.error(e.message),
	});

	const openEdit = (a: A2AAgent) => {
		setEdit(a);
		setEditForm({
			alias: a.alias,
			name: a.name,
			upstream_url: a.upstream_url,
			card_url: a.card_url ?? "",
			card_cache: formatCardCache(a.card_cache),
			api_key_env: a.api_key_env ?? "",
			auth_scheme: a.auth_scheme ?? "",
			enabled: a.enabled,
		});
		setEditError(null);
	};

	const list = agents.data ?? [];

	return (
		<PageBody>
			<PageHeader
				title="A2A"
				description="Proxy Agent2Agent agents through the gateway."
				info="Clients discover agents at /a2a/{alias}/.well-known/agent-card.json (URL rewritten to the gateway) and send JSON-RPC to POST /a2a/{alias}."
				actions={
					isOrgAdmin ? (
						<Button
							onClick={() => {
								setForm(blankForm());
								setError(null);
								setCreateOpen(true);
							}}
							disabled={!orgId}
						>
							<PlusIcon />
							Add agent
						</Button>
					) : null
				}
			/>
			<QueryGate
				isPending={agents.isPending || members.isPending}
				isError={agents.isError}
				error={agents.error}
				onRetry={() => agents.refetch()}
			>
				{list.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<BotIcon />
							</EmptyMedia>
							<EmptyTitle>No A2A agents</EmptyTitle>
							<EmptyDescription>
								Register a remote A2A agent to expose it at
								/a2a/&#123;alias&#125;.
								{!isOrgAdmin
									? " Only organization owners and admins can manage agents."
									: ""}
							</EmptyDescription>
						</EmptyHeader>
						{isOrgAdmin ? (
							<EmptyContent>
								<Button
									onClick={() => {
										setForm(blankForm());
										setError(null);
										setCreateOpen(true);
									}}
									disabled={!orgId}
								>
									<PlusIcon />
									Add agent
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<>
						{!isOrgAdmin ? (
							<p className="text-muted-foreground text-sm mb-3">
								Only organization owners and admins can create or edit A2A
								agents.
							</p>
						) : null}
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Alias</TableHead>
										<TableHead>Name</TableHead>
										<TableHead>Upstream</TableHead>
										<TableHead>Status</TableHead>
										<TableHead className="w-40" />
									</TableRow>
								</TableHeader>
								<TableBody>
									{list.map((a) => (
										<TableRow key={a.id}>
											<TableCell className="font-mono text-sm">
												{a.alias}
											</TableCell>
											<TableCell>{a.name}</TableCell>
											<TableCell className="max-w-xs truncate font-mono text-xs text-muted-foreground">
												{a.upstream_url}
											</TableCell>
											<TableCell>
												<Badge
													variant={a.enabled ? "secondary" : "outline"}
													className="text-xs font-normal"
												>
													{a.enabled ? "Enabled" : "Disabled"}
												</Badge>
											</TableCell>
											<TableCell className="text-right">
												{isOrgAdmin ? (
													<div className="flex justify-end gap-2">
														<Button
															variant="outline"
															size="sm"
															onClick={() => openEdit(a)}
														>
															Edit
														</Button>
														<Button
															variant="destructive"
															size="sm"
															disabled={del.isPending}
															onClick={() => {
																if (confirm(`Delete A2A agent “${a.alias}”?`)) {
																	del.mutate(a.id, {
																		onError: (e) => toast.error(e.message),
																	});
																}
															}}
														>
															Delete
														</Button>
													</div>
												) : null}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
					</>
				)}
			</QueryGate>

			<Sheet open={createOpen} onOpenChange={setCreateOpen}>
				<SheetContent className="overflow-y-auto sm:max-w-md">
					<SheetHeader>
						<SheetTitle>Add A2A agent</SheetTitle>
						<SheetDescription>
							Alias becomes /a2a/&#123;alias&#125; on the gateway.
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-col gap-4 px-4"
						onSubmit={(e) => {
							e.preventDefault();
							setError(null);
							if (
								!form.alias.trim() ||
								!form.name.trim() ||
								!form.upstream_url.trim()
							) {
								setError("Alias, name, and upstream URL are required.");
								return;
							}
							let card_cache: unknown | undefined;
							try {
								card_cache = parseCardCache(form.card_cache);
							} catch {
								setError("Card cache must be valid JSON.");
								return;
							}
							create.mutate(
								{
									orgId,
									alias: form.alias.trim(),
									name: form.name.trim(),
									upstream_url: form.upstream_url.trim(),
									card_url: form.card_url.trim() || undefined,
									card_cache,
									api_key_env: form.api_key_env.trim() || undefined,
									auth_scheme: form.auth_scheme.trim() || undefined,
									enabled: form.enabled,
								},
								{ onError: (err) => setError(err.message) },
							);
						}}
					>
						<div className="grid gap-2">
							<Label htmlFor="a2a-alias">Alias</Label>
							<Input
								id="a2a-alias"
								placeholder="helper"
								value={form.alias}
								onChange={(e) =>
									setForm((f) => ({ ...f, alias: e.target.value }))
								}
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="a2a-name">Name</Label>
							<Input
								id="a2a-name"
								placeholder="Helper agent"
								value={form.name}
								onChange={(e) =>
									setForm((f) => ({ ...f, name: e.target.value }))
								}
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="a2a-upstream">Upstream URL</Label>
							<Input
								id="a2a-upstream"
								placeholder="https://agent.example/rpc"
								value={form.upstream_url}
								onChange={(e) =>
									setForm((f) => ({ ...f, upstream_url: e.target.value }))
								}
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="a2a-card">Card URL (optional)</Label>
							<Input
								id="a2a-card"
								placeholder="https://agent.example/.well-known/agent-card.json"
								value={form.card_url}
								onChange={(e) =>
									setForm((f) => ({ ...f, card_url: e.target.value }))
								}
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="a2a-cache">Card cache JSON (optional)</Label>
							<Textarea
								id="a2a-cache"
								placeholder='{"name":"Helper","url":"..."}'
								value={form.card_cache}
								onChange={(e) =>
									setForm((f) => ({ ...f, card_cache: e.target.value }))
								}
								rows={4}
								className="font-mono text-xs"
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="a2a-env">API key env (optional)</Label>
							<Input
								id="a2a-env"
								placeholder="A2A_API_KEY"
								value={form.api_key_env}
								onChange={(e) =>
									setForm((f) => ({ ...f, api_key_env: e.target.value }))
								}
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="a2a-auth">Auth scheme (optional)</Label>
							<Input
								id="a2a-auth"
								placeholder="bearer"
								value={form.auth_scheme}
								onChange={(e) =>
									setForm((f) => ({ ...f, auth_scheme: e.target.value }))
								}
							/>
						</div>
						<div className="flex items-center gap-2 text-sm">
							<Checkbox
								id="a2a-enabled"
								checked={form.enabled}
								onCheckedChange={(v) =>
									setForm((f) => ({ ...f, enabled: v === true }))
								}
							/>
							<Label htmlFor="a2a-enabled">Enabled</Label>
						</div>
						{error ? <p className="text-destructive text-xs">{error}</p> : null}
						<SheetFooter className="gap-2 sm:flex-col sm:space-x-0">
							<Button
								type="button"
								variant="outline"
								disabled={
									testConn.isPending || !orgId || !form.upstream_url.trim()
								}
								onClick={() => {
									setError(null);
									if (!form.upstream_url.trim()) {
										setError("Upstream URL is required to test.");
										return;
									}
									testConn.mutate({
										orgId,
										upstream_url: form.upstream_url.trim(),
										card_url: form.card_url.trim() || undefined,
										api_key_env: form.api_key_env.trim() || undefined,
									});
								}}
							>
								{testConn.isPending ? "Testing…" : "Test connection"}
							</Button>
							<Button type="submit" disabled={create.isPending || !orgId}>
								{create.isPending ? "Creating…" : "Create"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>

			<Sheet open={!!edit} onOpenChange={(o) => !o && setEdit(null)}>
				<SheetContent className="overflow-y-auto sm:max-w-md">
					<SheetHeader>
						<SheetTitle>Edit A2A agent</SheetTitle>
						<SheetDescription>
							Changes publish into the next gateway snapshot.
						</SheetDescription>
					</SheetHeader>
					{edit ? (
						<form
							className="flex flex-col gap-4 px-4"
							onSubmit={(e) => {
								e.preventDefault();
								setEditError(null);
								if (
									!editForm.alias.trim() ||
									!editForm.name.trim() ||
									!editForm.upstream_url.trim()
								) {
									setEditError("Alias, name, and upstream URL are required.");
									return;
								}
								let card_cache: unknown | undefined;
								try {
									card_cache = parseCardCache(editForm.card_cache);
								} catch {
									setEditError("Card cache must be valid JSON.");
									return;
								}
								update.mutate(
									{
										agentId: edit.id,
										alias: editForm.alias.trim(),
										name: editForm.name.trim(),
										upstream_url: editForm.upstream_url.trim(),
										card_url: editForm.card_url.trim(),
										card_cache: card_cache ?? null,
										api_key_env: editForm.api_key_env.trim(),
										auth_scheme: editForm.auth_scheme.trim(),
										enabled: editForm.enabled,
									},
									{ onError: (err) => setEditError(err.message) },
								);
							}}
						>
							<div className="grid gap-2">
								<Label htmlFor="a2a-edit-alias">Alias</Label>
								<Input
									id="a2a-edit-alias"
									value={editForm.alias}
									onChange={(e) =>
										setEditForm((f) => ({ ...f, alias: e.target.value }))
									}
								/>
							</div>
							<div className="grid gap-2">
								<Label htmlFor="a2a-edit-name">Name</Label>
								<Input
									id="a2a-edit-name"
									value={editForm.name}
									onChange={(e) =>
										setEditForm((f) => ({ ...f, name: e.target.value }))
									}
								/>
							</div>
							<div className="grid gap-2">
								<Label htmlFor="a2a-edit-upstream">Upstream URL</Label>
								<Input
									id="a2a-edit-upstream"
									value={editForm.upstream_url}
									onChange={(e) =>
										setEditForm((f) => ({
											...f,
											upstream_url: e.target.value,
										}))
									}
								/>
							</div>
							<div className="grid gap-2">
								<Label htmlFor="a2a-edit-card">Card URL</Label>
								<Input
									id="a2a-edit-card"
									value={editForm.card_url}
									onChange={(e) =>
										setEditForm((f) => ({ ...f, card_url: e.target.value }))
									}
								/>
							</div>
							<div className="grid gap-2">
								<Label htmlFor="a2a-edit-cache">Card cache JSON</Label>
								<Textarea
									id="a2a-edit-cache"
									value={editForm.card_cache}
									onChange={(e) =>
										setEditForm((f) => ({
											...f,
											card_cache: e.target.value,
										}))
									}
									rows={4}
									className="font-mono text-xs"
								/>
							</div>
							<div className="grid gap-2">
								<Label htmlFor="a2a-edit-env">API key env</Label>
								<Input
									id="a2a-edit-env"
									value={editForm.api_key_env}
									onChange={(e) =>
										setEditForm((f) => ({
											...f,
											api_key_env: e.target.value,
										}))
									}
								/>
							</div>
							<div className="grid gap-2">
								<Label htmlFor="a2a-edit-auth">Auth scheme</Label>
								<Input
									id="a2a-edit-auth"
									value={editForm.auth_scheme}
									onChange={(e) =>
										setEditForm((f) => ({
											...f,
											auth_scheme: e.target.value,
										}))
									}
								/>
							</div>
							<div className="flex items-center gap-2 text-sm">
								<Checkbox
									id="a2a-edit-enabled"
									checked={editForm.enabled}
									onCheckedChange={(v) =>
										setEditForm((f) => ({ ...f, enabled: v === true }))
									}
								/>
								<Label htmlFor="a2a-edit-enabled">Enabled</Label>
							</div>
							{editError ? (
								<p className="text-destructive text-xs">{editError}</p>
							) : null}
							<SheetFooter className="gap-2 sm:flex-col sm:space-x-0">
								<Button
									type="button"
									variant="outline"
									disabled={
										testConn.isPending ||
										!orgId ||
										!editForm.upstream_url.trim()
									}
									onClick={() => {
										setEditError(null);
										if (!editForm.upstream_url.trim()) {
											setEditError("Upstream URL is required to test.");
											return;
										}
										testConn.mutate({
											orgId,
											upstream_url: editForm.upstream_url.trim(),
											card_url: editForm.card_url.trim() || undefined,
											api_key_env: editForm.api_key_env.trim() || undefined,
										});
									}}
								>
									{testConn.isPending ? "Testing…" : "Test connection"}
								</Button>
								<Button type="submit" disabled={update.isPending}>
									{update.isPending ? "Saving…" : "Save"}
								</Button>
							</SheetFooter>
						</form>
					) : null}
				</SheetContent>
			</Sheet>
		</PageBody>
	);
}
