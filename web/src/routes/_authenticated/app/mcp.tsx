import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { CableIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import {
	createMCPBackendMutationOptions,
	deleteMCPBackendMutationOptions,
	type MCPBackend,
	mcpBackendsQueryOptions,
	type ProtocolProbeResult,
	testMCPBackendMutationOptions,
	updateMCPBackendMutationOptions,
} from "#/api/mcp";
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
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/mcp")({
	...pageTitle("MCP"),
	component: RouteComponent,
});

function parseAllowlist(raw: string): string[] {
	return raw
		.split(/[\n,]+/)
		.map((s) => s.trim())
		.filter(Boolean);
}

function formatAllowlist(list: string[] | string | null | undefined): string {
	if (list == null || list === "") return "";
	if (typeof list === "string") {
		try {
			const parsed = JSON.parse(list) as unknown;
			if (Array.isArray(parsed)) {
				return parsed.map(String).join(", ");
			}
		} catch {
			return list;
		}
		return list;
	}
	return list.join(", ");
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

function blankForm() {
	return {
		alias: "",
		name: "",
		base_url: "",
		api_key_env: "",
		method_allowlist: "",
		enabled: true,
	};
}

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const backends = useQuery(mcpBackendsQueryOptions(orgId));
	const members = useQuery(orgMembersQueryOptions(orgId));
	const [createOpen, setCreateOpen] = useState(false);
	const [edit, setEdit] = useState<MCPBackend | null>(null);
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
			queryKey: ["organizations", orgId, "mcp-backends"],
		});

	const create = useMutation({
		...createMCPBackendMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("MCP backend created");
			setCreateOpen(false);
			setForm(blankForm());
		},
	});
	const update = useMutation({
		...updateMCPBackendMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("MCP backend updated");
			setEdit(null);
		},
	});
	const del = useMutation({
		...deleteMCPBackendMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("MCP backend deleted");
		},
	});
	const testConn = useMutation({
		...testMCPBackendMutationOptions(),
		onSuccess: (res) => probeToast(res),
		onError: (e) => toast.error(e.message),
	});

	const openEdit = (b: MCPBackend) => {
		setEdit(b);
		setEditForm({
			alias: b.alias,
			name: b.name,
			base_url: b.base_url,
			api_key_env: b.api_key_env ?? "",
			method_allowlist: formatAllowlist(b.method_allowlist),
			enabled: b.enabled,
		});
		setEditError(null);
	};

	const list = backends.data ?? [];

	return (
		<PageBody>
			<PageHeader
				title="MCP"
				description="Proxy Streamable HTTP MCP servers through the gateway."
				info="Clients call POST|GET|DELETE /mcp/{alias} with a virtual API key. The gateway authenticates, applies quotas/policies, and forwards to the upstream base URL."
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
							Add backend
						</Button>
					) : null
				}
			/>
			<QueryGate
				isPending={backends.isPending || members.isPending}
				isError={backends.isError}
				error={backends.error}
				onRetry={() => backends.refetch()}
			>
				{list.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<CableIcon />
							</EmptyMedia>
							<EmptyTitle>No MCP backends</EmptyTitle>
							<EmptyDescription>
								Register a remote MCP server to expose it at
								/mcp/&#123;alias&#125;.
								{!isOrgAdmin
									? " Only organization owners and admins can manage backends."
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
									Add backend
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<>
						{!isOrgAdmin ? (
							<p className="text-muted-foreground text-sm mb-3">
								Only organization owners and admins can create or edit MCP
								backends.
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
									{list.map((b) => (
										<TableRow key={b.id}>
											<TableCell className="font-mono text-sm">
												{b.alias}
											</TableCell>
											<TableCell>{b.name}</TableCell>
											<TableCell className="max-w-xs truncate font-mono text-xs text-muted-foreground">
												{b.base_url}
											</TableCell>
											<TableCell>
												<Badge
													variant={b.enabled ? "secondary" : "outline"}
													className="text-xs font-normal"
												>
													{b.enabled ? "Enabled" : "Disabled"}
												</Badge>
											</TableCell>
											<TableCell className="text-right">
												{isOrgAdmin ? (
													<div className="flex justify-end gap-2">
														<Button
															variant="outline"
															size="sm"
															onClick={() => openEdit(b)}
														>
															Edit
														</Button>
														<Button
															variant="destructive"
															size="sm"
															disabled={del.isPending}
															onClick={() => {
																if (
																	confirm(`Delete MCP backend “${b.alias}”?`)
																) {
																	del.mutate(b.id, {
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
						<SheetTitle>Add MCP backend</SheetTitle>
						<SheetDescription>
							Alias becomes the gateway path /mcp/&#123;alias&#125;.
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
								!form.base_url.trim()
							) {
								setError("Alias, name, and base URL are required.");
								return;
							}
							create.mutate(
								{
									orgId,
									alias: form.alias.trim(),
									name: form.name.trim(),
									base_url: form.base_url.trim(),
									api_key_env: form.api_key_env.trim() || undefined,
									method_allowlist: parseAllowlist(form.method_allowlist),
									enabled: form.enabled,
								},
								{ onError: (err) => setError(err.message) },
							);
						}}
					>
						<div className="grid gap-2">
							<Label htmlFor="mcp-alias">Alias</Label>
							<Input
								id="mcp-alias"
								placeholder="docs"
								value={form.alias}
								onChange={(e) =>
									setForm((f) => ({ ...f, alias: e.target.value }))
								}
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="mcp-name">Name</Label>
							<Input
								id="mcp-name"
								placeholder="Docs MCP"
								value={form.name}
								onChange={(e) =>
									setForm((f) => ({ ...f, name: e.target.value }))
								}
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="mcp-base">Base URL</Label>
							<Input
								id="mcp-base"
								placeholder="https://mcp.example.com/mcp"
								value={form.base_url}
								onChange={(e) =>
									setForm((f) => ({ ...f, base_url: e.target.value }))
								}
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="mcp-env">API key env (optional)</Label>
							<Input
								id="mcp-env"
								placeholder="MCP_API_KEY"
								value={form.api_key_env}
								onChange={(e) =>
									setForm((f) => ({ ...f, api_key_env: e.target.value }))
								}
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="mcp-allow">Method allowlist (optional)</Label>
							<Input
								id="mcp-allow"
								placeholder="tools/list, tools/call"
								value={form.method_allowlist}
								onChange={(e) =>
									setForm((f) => ({
										...f,
										method_allowlist: e.target.value,
									}))
								}
							/>
							<p className="text-muted-foreground text-xs">
								Comma-separated JSON-RPC methods. Empty allows all.
							</p>
						</div>
						<div className="flex items-center gap-2 text-sm">
							<Checkbox
								id="mcp-enabled"
								checked={form.enabled}
								onCheckedChange={(v) =>
									setForm((f) => ({ ...f, enabled: v === true }))
								}
							/>
							<Label htmlFor="mcp-enabled">Enabled</Label>
						</div>
						{error ? <p className="text-destructive text-xs">{error}</p> : null}
						<SheetFooter className="gap-2 sm:flex-col sm:space-x-0">
							<Button
								type="button"
								variant="outline"
								disabled={testConn.isPending || !orgId || !form.base_url.trim()}
								onClick={() => {
									setError(null);
									if (!form.base_url.trim()) {
										setError("Base URL is required to test.");
										return;
									}
									testConn.mutate({
										orgId,
										base_url: form.base_url.trim(),
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
						<SheetTitle>Edit MCP backend</SheetTitle>
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
									!editForm.base_url.trim()
								) {
									setEditError("Alias, name, and base URL are required.");
									return;
								}
								update.mutate(
									{
										backendId: edit.id,
										alias: editForm.alias.trim(),
										name: editForm.name.trim(),
										base_url: editForm.base_url.trim(),
										api_key_env: editForm.api_key_env.trim(),
										method_allowlist: parseAllowlist(editForm.method_allowlist),
										enabled: editForm.enabled,
									},
									{ onError: (err) => setEditError(err.message) },
								);
							}}
						>
							<div className="grid gap-2">
								<Label htmlFor="mcp-edit-alias">Alias</Label>
								<Input
									id="mcp-edit-alias"
									value={editForm.alias}
									onChange={(e) =>
										setEditForm((f) => ({ ...f, alias: e.target.value }))
									}
								/>
							</div>
							<div className="grid gap-2">
								<Label htmlFor="mcp-edit-name">Name</Label>
								<Input
									id="mcp-edit-name"
									value={editForm.name}
									onChange={(e) =>
										setEditForm((f) => ({ ...f, name: e.target.value }))
									}
								/>
							</div>
							<div className="grid gap-2">
								<Label htmlFor="mcp-edit-base">Base URL</Label>
								<Input
									id="mcp-edit-base"
									value={editForm.base_url}
									onChange={(e) =>
										setEditForm((f) => ({ ...f, base_url: e.target.value }))
									}
								/>
							</div>
							<div className="grid gap-2">
								<Label htmlFor="mcp-edit-env">API key env</Label>
								<Input
									id="mcp-edit-env"
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
								<Label htmlFor="mcp-edit-allow">Method allowlist</Label>
								<Input
									id="mcp-edit-allow"
									value={editForm.method_allowlist}
									onChange={(e) =>
										setEditForm((f) => ({
											...f,
											method_allowlist: e.target.value,
										}))
									}
								/>
							</div>
							<div className="flex items-center gap-2 text-sm">
								<Checkbox
									id="mcp-edit-enabled"
									checked={editForm.enabled}
									onCheckedChange={(v) =>
										setEditForm((f) => ({ ...f, enabled: v === true }))
									}
								/>
								<Label htmlFor="mcp-edit-enabled">Enabled</Label>
							</div>
							{editError ? (
								<p className="text-destructive text-xs">{editError}</p>
							) : null}
							<SheetFooter className="gap-2 sm:flex-col sm:space-x-0">
								<Button
									type="button"
									variant="outline"
									disabled={
										testConn.isPending || !orgId || !editForm.base_url.trim()
									}
									onClick={() => {
										setEditError(null);
										if (!editForm.base_url.trim()) {
											setEditError("Base URL is required to test.");
											return;
										}
										testConn.mutate({
											orgId,
											base_url: editForm.base_url.trim(),
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
