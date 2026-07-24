import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { PlusIcon, RouteIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import {
	orgDefaultRetryQueryOptions,
	orgMembersQueryOptions,
} from "#/api/organization";
import { providersQueryOptions } from "#/api/provider";
import {
	createRouteMutationOptions,
	deleteRouteMutationOptions,
	formatRoutingStrategy,
	parseRoutingStrategy,
	type RetryConfig,
	type RouteConfig,
	type RoutingStrategy,
	routesQueryOptions,
	updateRouteMutationOptions,
} from "#/api/routing";
import { InfoAlert } from "#/components/info-alert";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import {
	FallbackList,
	type FallbackRow,
} from "#/components/routing/fallback-list";
import {
	formatRetrySummary,
	RetryEditor,
	toRetryPayload,
	validateRetry,
} from "#/components/routing/retry-editor";
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

export const Route = createFileRoute("/_authenticated/app/routing")({
	...pageTitle("Routing"),
	component: RouteComponent,
});

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const routes = useQuery(routesQueryOptions(orgId));
	const providers = useQuery(providersQueryOptions(orgId));
	const members = useQuery(orgMembersQueryOptions(orgId));
	const orgRetry = useQuery(orgDefaultRetryQueryOptions(orgId));
	const [createOpen, setCreateOpen] = useState(false);
	const [edit, setEdit] = useState<RouteConfig | null>(null);

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const create = useMutation({
		...createRouteMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "routes"],
			});
			toast.success("Route created");
			setCreateOpen(false);
		},
	});
	const update = useMutation({
		...updateRouteMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "routes"],
			});
			toast.success("Route updated");
			setEdit(null);
		},
	});
	const del = useMutation({
		...deleteRouteMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "routes"],
			});
			toast.success("Route deleted");
		},
	});

	const [model, setModel] = useState("ping-model");
	const [targetModel, setTargetModel] = useState("gpt-4o-mini");
	const [providerId, setProviderId] = useState("");
	const [fallbacks, setFallbacks] = useState<FallbackRow[]>([]);
	const [retry, setRetry] = useState<RetryConfig | null>(null);
	const [strategy, setStrategy] = useState<RoutingStrategy>("ordered");
	const [weight, setWeight] = useState(1);
	const [error, setError] = useState<string | null>(null);

	const [editModel, setEditModel] = useState("");
	const [editTargetModel, setEditTargetModel] = useState("");
	const [editProviderId, setEditProviderId] = useState("");
	const [editFallbacks, setEditFallbacks] = useState<FallbackRow[]>([]);
	const [editRetry, setEditRetry] = useState<RetryConfig | null>(null);
	const [editStrategy, setEditStrategy] = useState<RoutingStrategy>("ordered");
	const [editWeight, setEditWeight] = useState(1);
	const [editError, setEditError] = useState<string | null>(null);

	const openCreate = () => {
		setRetry(null);
		setStrategy("ordered");
		setWeight(1);
		setError(null);
		setCreateOpen(true);
	};

	const openEdit = (r: RouteConfig) => {
		setEdit(r);
		setEditModel(r.model);
		setEditTargetModel(r.target_model);
		setEditProviderId(r.provider_id);
		setEditFallbacks(
			(r.fallbacks ?? []).map((f) => ({
				...f,
				weight: f.weight ?? 1,
				key: crypto.randomUUID(),
			})),
		);
		setEditRetry(r.retry ?? null);
		setEditStrategy(parseRoutingStrategy(r.routing_strategy));
		setEditWeight(r.weight && r.weight > 0 ? r.weight : 1);
		setEditError(null);
	};

	const providerList = providers.data ?? [];
	const routeList = routes.data ?? [];
	const selectedProvider = providerId || providerList[0]?.id || "";
	const providerName = (id: string) =>
		providerList.find((p) => p.id === id)?.name ?? id;

	const canAddRoute = isOrgAdmin && !!orgId && providerList.length > 0;

	return (
		<PageBody>
			<PageHeader
				title="Routing"
				description="Map requested model names to providers."
				info="Optional route retries override the org default. Strategies: ordered (config order), weighted (random first pick), latency (gateway EWMA), cost (catalog unit price). Failover on 5xx/timeout/429."
				actions={
					isOrgAdmin ? (
						<Button onClick={openCreate} disabled={!canAddRoute}>
							<PlusIcon />
							Add route
						</Button>
					) : null
				}
			/>
			<QueryGate
				isPending={
					!!orgId &&
					(routes.isLoading || providers.isLoading || members.isPending)
				}
				isError={routes.isError || providers.isError}
				error={routes.error || providers.error}
				onRetry={() => {
					void routes.refetch();
					void providers.refetch();
				}}
			>
				{providerList.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<RouteIcon />
							</EmptyMedia>
							<EmptyTitle>Add a provider first</EmptyTitle>
							<EmptyDescription>
								Routes need an upstream provider. Create one, then come back
								here.
								{!isOrgAdmin
									? " Only organization owners and admins can manage routing."
									: ""}
							</EmptyDescription>
						</EmptyHeader>
						<EmptyContent>
							<Button
								nativeButton={false}
								render={<Link to="/app/providers" />}
							>
								Go to Providers
							</Button>
						</EmptyContent>
					</Empty>
				) : routeList.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<RouteIcon />
							</EmptyMedia>
							<EmptyTitle>No routes</EmptyTitle>
							<EmptyDescription>
								Create a virtual model name clients will request.
								{!isOrgAdmin
									? " Only organization owners and admins can create routes."
									: ""}
							</EmptyDescription>
						</EmptyHeader>
						{isOrgAdmin ? (
							<EmptyContent>
								<Button onClick={openCreate}>
									<PlusIcon />
									Add route
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<>
						{!isOrgAdmin ? (
							<p className="text-muted-foreground text-sm">
								Only organization owners and admins can create or edit routes.
							</p>
						) : null}
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Model</TableHead>
									<TableHead>Target</TableHead>
									<TableHead>Provider</TableHead>
									<TableHead>Strategy</TableHead>
									<TableHead>Retry</TableHead>
									<TableHead>Fallbacks</TableHead>
									{isOrgAdmin ? <TableHead className="w-40" /> : null}
								</TableRow>
							</TableHeader>
							<TableBody>
								{routeList.map((r) => (
									<TableRow key={r.id}>
										<TableCell className="font-medium">{r.model}</TableCell>
										<TableCell className="font-mono text-xs">
											{r.target_model}
										</TableCell>
										<TableCell>
											<div className="text-sm">
												{providerName(r.provider_id)}
											</div>
											<div className="text-muted-foreground font-mono text-xs">
												{r.provider_id}
											</div>
										</TableCell>
										<TableCell className="text-muted-foreground text-xs">
											{formatRoutingStrategy(r.routing_strategy, r.weight)}
										</TableCell>
										<TableCell className="text-muted-foreground text-xs">
											{formatRetrySummary(r.retry, orgRetry.data?.retry)}
										</TableCell>
										<TableCell>
											{(r.fallbacks ?? []).length === 0 ? (
												<span className="text-muted-foreground text-xs">—</span>
											) : (
												<div className="flex flex-wrap gap-1">
													{(r.fallbacks ?? []).map((f) => (
														<Badge
															key={`${f.provider_id}:${f.target_model}`}
															variant="outline"
															className="text-xs font-normal"
														>
															{providerName(f.provider_id)} →{" "}
															{f.target_model || r.target_model}
															{r.routing_strategy === "weighted"
																? ` · w${f.weight ?? 1}`
																: ""}
														</Badge>
													))}
												</div>
											)}
										</TableCell>
										{isOrgAdmin ? (
											<TableCell className="space-x-2">
												<Button
													variant="outline"
													size="sm"
													onClick={() => openEdit(r)}
												>
													Edit
												</Button>
												<Button
													variant="outline"
													size="sm"
													disabled={del.isPending}
													onClick={() => del.mutate(r.id)}
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
				<SheetContent className="w-full overflow-y-auto sm:max-w-2xl data-[side=right]:sm:max-w-2xl data-[side=left]:sm:max-w-2xl">
					<SheetHeader>
						<SheetTitle>Add route</SheetTitle>
						<SheetDescription>
							Add a model mapping for this organization.
						</SheetDescription>
						<InfoAlert>
							Publishes a new gateway snapshot with this model mapping.
						</InfoAlert>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 overflow-y-auto px-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!orgId || !selectedProvider) return;
							const retryErr = validateRetry(retry);
							if (retryErr) {
								setError(retryErr);
								return;
							}
							setError(null);
							create.mutate(
								{
									orgId,
									model,
									provider_id: selectedProvider,
									target_model: targetModel || model,
									fallbacks: fallbacks
										.filter((f) => f.provider_id)
										.map(({ provider_id, target_model, weight: w }) => ({
											provider_id,
											target_model,
											weight: w && w > 0 ? w : 1,
										})),
									retry: toRetryPayload(retry),
									routing_strategy: strategy,
									weight: weight > 0 ? weight : 1,
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
							<Label htmlFor="route-model">Requested model</Label>
							<Input
								id="route-model"
								value={model}
								onChange={(e) => setModel(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="route-target">Target model</Label>
							<Input
								id="route-target"
								value={targetModel}
								onChange={(e) => setTargetModel(e.target.value)}
								placeholder={model}
							/>
						</div>
						<div className="space-y-1">
							<Label>Provider</Label>
							<Select
								value={selectedProvider}
								onValueChange={(v) => setProviderId(v ?? "")}
							>
								<SelectTrigger className="w-full">
									<SelectValue placeholder="Select provider" />
								</SelectTrigger>
								<SelectContent>
									{providerList.map((p) => (
										<SelectItem key={p.id} value={p.id}>
											{p.name} ({p.type})
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1">
							<Label>Routing strategy</Label>
							<Select
								value={strategy}
								onValueChange={(v) => setStrategy(parseRoutingStrategy(v))}
							>
								<SelectTrigger className="w-full">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="ordered">
										Ordered (failover list)
									</SelectItem>
									<SelectItem value="weighted">
										Weighted load balance
									</SelectItem>
									<SelectItem value="latency">
										Latency (EWMA)
									</SelectItem>
									<SelectItem value="cost">Cost (catalog)</SelectItem>
								</SelectContent>
							</Select>
						</div>
						{strategy === "weighted" ? (
							<div className="space-y-1">
								<Label htmlFor="route-weight">Primary weight</Label>
								<Input
									id="route-weight"
									type="number"
									min={1}
									step={1}
									value={weight}
									onChange={(e) => {
										const n = Number.parseInt(e.target.value, 10);
										setWeight(Number.isFinite(n) && n > 0 ? n : 1);
									}}
								/>
							</div>
						) : null}
						<RetryEditor
							value={retry}
							onChange={setRetry}
							idPrefix="create-retry"
						/>
						<FallbackList
							fallbacks={fallbacks}
							onChange={setFallbacks}
							providers={providerList}
							defaultTargetModel={targetModel || model}
							showWeights={strategy === "weighted"}
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
								disabled={create.isPending || !orgId || !selectedProvider}
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
						<SheetTitle>Edit route</SheetTitle>
						<SheetDescription>
							Update model mapping, retry, and fallbacks.
						</SheetDescription>
						<InfoAlert>Publishes a new gateway snapshot.</InfoAlert>
					</SheetHeader>
					{edit ? (
						<form
							className="flex flex-1 flex-col gap-4 overflow-y-auto px-4"
							onSubmit={(e) => {
								e.preventDefault();
								if (!editProviderId) return;
								const retryErr = validateRetry(editRetry);
								if (retryErr) {
									setEditError(retryErr);
									return;
								}
								setEditError(null);
								update.mutate(
									{
										routeId: edit.id,
										model: editModel,
										provider_id: editProviderId,
										target_model: editTargetModel || editModel,
										fallbacks: editFallbacks
											.filter((f) => f.provider_id)
											.map(({ provider_id, target_model, weight: w }) => ({
												provider_id,
												target_model,
												weight: w && w > 0 ? w : 1,
											})),
										retry: toRetryPayload(editRetry),
										routing_strategy: editStrategy,
										weight: editWeight > 0 ? editWeight : 1,
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
								<Label htmlFor="edit-route-model">Requested model</Label>
								<Input
									id="edit-route-model"
									value={editModel}
									onChange={(e) => setEditModel(e.target.value)}
									required
								/>
							</div>
							<div className="space-y-1">
								<Label htmlFor="edit-route-target">Target model</Label>
								<Input
									id="edit-route-target"
									value={editTargetModel}
									onChange={(e) => setEditTargetModel(e.target.value)}
									placeholder={editModel}
								/>
							</div>
							<div className="space-y-1">
								<Label>Provider</Label>
								<Select
									value={editProviderId}
									onValueChange={(v) => setEditProviderId(v ?? "")}
								>
									<SelectTrigger className="w-full">
										<SelectValue placeholder="Select provider" />
									</SelectTrigger>
									<SelectContent>
										{providerList.map((p) => (
											<SelectItem key={p.id} value={p.id}>
												{p.name} ({p.type})
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
							<div className="space-y-1">
								<Label>Routing strategy</Label>
								<Select
									value={editStrategy}
									onValueChange={(v) =>
										setEditStrategy(parseRoutingStrategy(v))
									}
								>
									<SelectTrigger className="w-full">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="ordered">
											Ordered (failover list)
										</SelectItem>
										<SelectItem value="weighted">
											Weighted load balance
										</SelectItem>
										<SelectItem value="latency">
											Latency (EWMA)
										</SelectItem>
										<SelectItem value="cost">Cost (catalog)</SelectItem>
									</SelectContent>
								</Select>
							</div>
							{editStrategy === "weighted" ? (
								<div className="space-y-1">
									<Label htmlFor="edit-route-weight">Primary weight</Label>
									<Input
										id="edit-route-weight"
										type="number"
										min={1}
										step={1}
										value={editWeight}
										onChange={(e) => {
											const n = Number.parseInt(e.target.value, 10);
											setEditWeight(Number.isFinite(n) && n > 0 ? n : 1);
										}}
									/>
								</div>
							) : null}
							<RetryEditor
								value={editRetry}
								onChange={setEditRetry}
								idPrefix="edit-retry"
							/>
							<FallbackList
								fallbacks={editFallbacks}
								onChange={setEditFallbacks}
								providers={providerList}
								defaultTargetModel={editTargetModel || editModel}
								showWeights={editStrategy === "weighted"}
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
									disabled={update.isPending || !editProviderId}
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
