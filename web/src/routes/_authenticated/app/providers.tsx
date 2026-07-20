import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { PlugIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { orgMembersQueryOptions } from "#/api/organization";
import {
	createProviderMutationOptions,
	deleteProviderMutationOptions,
	PROVIDER_TYPE_PRESETS,
	type Provider,
	type ProviderHealth,
	type ProviderHealthStatus,
	providerHealthQueryOptions,
	providersQueryOptions,
	updateProviderMutationOptions,
} from "#/api/provider";
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

export const Route = createFileRoute("/_authenticated/app/providers")({
	...pageTitle("Providers"),
	component: RouteComponent,
});

function CapChips({ p }: { p: Provider }) {
	const caps = p.capabilities ?? { chat: false, stream: false };
	const items: string[] = [];
	if (caps.chat) items.push("chat");
	if (caps.stream) items.push("stream");
	if (caps.tts) items.push("tts");
	if (caps.stt) items.push("stt");
	if (items.length === 0) items.push("—");
	return (
		<div className="flex flex-wrap gap-1">
			{items.map((c) => (
				<Badge key={c} variant="outline" className="text-xs font-normal">
					{c}
				</Badge>
			))}
		</div>
	);
}

function HealthChip({ h }: { h?: ProviderHealth }) {
	const status: ProviderHealthStatus = h?.status ?? "unknown";
	const label =
		status === "healthy"
			? "Healthy"
			: status === "degraded"
				? "Degraded"
				: status === "down"
					? "Down"
					: "Unknown";
	const tip = h
		? `${h.requests} req · ${Math.round(h.error_rate * 100)}% err · ${Math.round(h.avg_latency_ms)}ms avg (24h)`
		: "No traffic in the last 24h";
	return (
		<span title={tip}>
			<Badge
				variant={
					status === "healthy"
						? "secondary"
						: status === "unknown"
							? "outline"
							: "destructive"
				}
				className="text-xs font-normal"
			>
				{label}
			</Badge>
		</span>
	);
}

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const providers = useQuery(providersQueryOptions(orgId));
	const health = useQuery(providerHealthQueryOptions(orgId));
	const members = useQuery(orgMembersQueryOptions(orgId));
	const healthById = new Map(
		(health.data ?? []).map((h) => [h.provider_id, h] as const),
	);

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const [createOpen, setCreateOpen] = useState(false);
	const [edit, setEdit] = useState<Provider | null>(null);

	const create = useMutation({
		...createProviderMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "providers"],
			});
			toast.success("Provider created");
			setCreateOpen(false);
		},
	});
	const update = useMutation({
		...updateProviderMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "providers"],
			});
			toast.success("Provider updated");
			setEdit(null);
		},
	});
	const del = useMutation({
		...deleteProviderMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "providers"],
			});
			toast.success("Provider deleted");
		},
	});

	const preset = PROVIDER_TYPE_PRESETS.openai;
	const [name, setName] = useState(preset.name);
	const [type, setType] = useState("openai");
	const [baseURL, setBaseURL] = useState(preset.base_url);
	const [apiKeyEnv, setApiKeyEnv] = useState(preset.api_key_env);
	const [error, setError] = useState<string | null>(null);

	const [editName, setEditName] = useState("");
	const [editBase, setEditBase] = useState("");
	const [editEnv, setEditEnv] = useState("");
	const [editError, setEditError] = useState<string | null>(null);

	const applyTypeDefaults = (next: string) => {
		setType(next);
		const p = PROVIDER_TYPE_PRESETS[next];
		if (!p) return;
		setName(p.name);
		setBaseURL(p.base_url);
		setApiKeyEnv(p.api_key_env);
	};

	const openEdit = (p: Provider) => {
		setEdit(p);
		setEditName(p.name);
		setEditBase(p.base_url);
		setEditEnv(p.api_key_env);
		setEditError(null);
	};

	const typeCaps = PROVIDER_TYPE_PRESETS[type]?.caps;

	return (
		<PageBody>
			<PageHeader
				title="Providers"
				description="Upstream LLM providers. Health is derived from usage over the last 24h (routed models). Credentials are environment variable references on the gateway."
				actions={
					isOrgAdmin ? (
						<Button onClick={() => setCreateOpen(true)} disabled={!orgId}>
							<PlusIcon />
							Add provider
						</Button>
					) : null
				}
			/>
			<QueryGate
				isPending={!!orgId && (providers.isLoading || members.isPending)}
				isError={providers.isError}
				error={providers.error}
				onRetry={() => {
					void providers.refetch();
					void health.refetch();
				}}
			>
				{(providers.data ?? []).length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<PlugIcon />
							</EmptyMedia>
							<EmptyTitle>No providers</EmptyTitle>
							<EmptyDescription>
								Add OpenAI, Anthropic, Gemini, or an OpenAI-compatible base URL.
								Then create routes under Routing.
								{!isOrgAdmin
									? " Only organization owners and admins can create providers."
									: ""}
							</EmptyDescription>
						</EmptyHeader>
						{isOrgAdmin ? (
							<EmptyContent>
								<Button onClick={() => setCreateOpen(true)}>
									<PlusIcon />
									Add provider
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<>
						{!isOrgAdmin ? (
							<p className="text-muted-foreground text-sm">
								Only organization owners and admins can create or edit
								providers.
							</p>
						) : null}
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Name</TableHead>
									<TableHead>Type</TableHead>
									<TableHead>Health</TableHead>
									<TableHead>Base URL</TableHead>
									<TableHead>Env</TableHead>
									<TableHead>Capabilities</TableHead>
									{isOrgAdmin ? <TableHead className="w-40" /> : null}
								</TableRow>
							</TableHeader>
							<TableBody>
								{(providers.data ?? []).map((p) => (
									<TableRow key={p.id}>
										<TableCell className="font-medium">{p.name}</TableCell>
										<TableCell>
											<Badge variant="secondary">{p.type}</Badge>
										</TableCell>
										<TableCell>
											<HealthChip h={healthById.get(p.id)} />
										</TableCell>
										<TableCell className="text-muted-foreground max-w-[14rem] truncate text-xs">
											{p.base_url}
										</TableCell>
										<TableCell className="font-mono text-xs">
											{p.api_key_env}
										</TableCell>
										<TableCell>
											<CapChips p={p} />
										</TableCell>
										{isOrgAdmin ? (
											<TableCell className="space-x-2">
												<Button
													variant="outline"
													size="sm"
													onClick={() => openEdit(p)}
												>
													Edit
												</Button>
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
					</>
				)}
			</QueryGate>

			<p className="text-muted-foreground mt-4 text-sm">
				After providers are ready, map models in{" "}
				<Link to="/app/routing" className="underline">
					Routing
				</Link>
				.
			</p>

			<Sheet open={createOpen} onOpenChange={setCreateOpen}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Add provider</SheetTitle>
						<SheetDescription>
							Publishes a new gateway snapshot. Set the env var on the gateway
							process.
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
									base_url: baseURL,
									api_key_env: apiKeyEnv,
									type,
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
							<Label>Type</Label>
							<Select
								value={type}
								onValueChange={(v) => applyTypeDefaults(v ?? "openai")}
							>
								<SelectTrigger className="w-full">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									{Object.keys(PROVIDER_TYPE_PRESETS).map((t) => (
										<SelectItem key={t} value={t}>
											{t}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
							{typeCaps ? (
								<p className="text-muted-foreground text-xs">
									Defaults:{" "}
									{[
										typeCaps.chat && "chat",
										typeCaps.stream && "stream",
										typeCaps.tts && "tts",
										typeCaps.stt && "stt",
									]
										.filter(Boolean)
										.join(", ")}
								</p>
							) : null}
						</div>
						<div className="space-y-1">
							<Label htmlFor="prov-name">Name</Label>
							<Input
								id="prov-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="prov-base">Base URL</Label>
							<Input
								id="prov-base"
								value={baseURL}
								onChange={(e) => setBaseURL(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="prov-env">API key env var</Label>
							<Input
								id="prov-env"
								value={apiKeyEnv}
								onChange={(e) => setApiKeyEnv(e.target.value)}
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

			<Sheet
				open={!!edit}
				onOpenChange={(o) => {
					if (!o) setEdit(null);
				}}
			>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Edit provider</SheetTitle>
						<SheetDescription>
							Type is fixed after create. Update name, base URL, or env ref.
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
										providerId: edit.id,
										name: editName,
										base_url: editBase,
										api_key_env: editEnv,
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
								<Label>Type</Label>
								<Input readOnly value={edit.type} className="bg-muted" />
							</div>
							<div className="space-y-1">
								<Label htmlFor="edit-name">Name</Label>
								<Input
									id="edit-name"
									value={editName}
									onChange={(e) => setEditName(e.target.value)}
									required
								/>
							</div>
							<div className="space-y-1">
								<Label htmlFor="edit-base">Base URL</Label>
								<Input
									id="edit-base"
									value={editBase}
									onChange={(e) => setEditBase(e.target.value)}
									required
								/>
							</div>
							<div className="space-y-1">
								<Label htmlFor="edit-env">API key env var</Label>
								<Input
									id="edit-env"
									value={editEnv}
									onChange={(e) => setEditEnv(e.target.value)}
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
