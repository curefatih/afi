import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { PlusIcon, RouteIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { orgMembersQueryOptions } from "#/api/organization";
import { providersQueryOptions } from "#/api/provider";
import {
	createRouteMutationOptions,
	deleteRouteMutationOptions,
	type RouteConfig,
	type RouteFallback,
	routesQueryOptions,
	updateRouteMutationOptions,
} from "#/api/routing";
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
	const [fallbacks, setFallbacks] = useState<
		Array<RouteFallback & { key: string }>
	>([]);
	const [error, setError] = useState<string | null>(null);

	const [editModel, setEditModel] = useState("");
	const [editTargetModel, setEditTargetModel] = useState("");
	const [editProviderId, setEditProviderId] = useState("");
	const [editFallbacks, setEditFallbacks] = useState<
		Array<RouteFallback & { key: string }>
	>([]);
	const [editError, setEditError] = useState<string | null>(null);

	const openEdit = (r: RouteConfig) => {
		setEdit(r);
		setEditModel(r.model);
		setEditTargetModel(r.target_model);
		setEditProviderId(r.provider_id);
		setEditFallbacks(
			(r.fallbacks ?? []).map((f) => ({
				...f,
				key: crypto.randomUUID(),
			})),
		);
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
				description="Map requested model names to providers. Optional fallbacks run on 5xx/timeout/429 before the response body is committed."
				actions={
					isOrgAdmin ? (
						<Button onClick={() => setCreateOpen(true)} disabled={!canAddRoute}>
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
								<Button onClick={() => setCreateOpen(true)}>
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
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Add route</SheetTitle>
						<SheetDescription>
							Publishes a new gateway snapshot with this model mapping.
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!orgId || !selectedProvider) return;
							setError(null);
							create.mutate(
								{
									orgId,
									model,
									provider_id: selectedProvider,
									target_model: targetModel || model,
									fallbacks: fallbacks
										.filter((f) => f.provider_id)
										.map(({ provider_id, target_model }) => ({
											provider_id,
											target_model,
										})),
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
						<div className="space-y-2">
							<div className="flex items-center justify-between">
								<Label>Fallbacks</Label>
								<Button
									type="button"
									variant="outline"
									size="sm"
									onClick={() =>
										setFallbacks((prev) => [
											...prev,
											{
												key: crypto.randomUUID(),
												provider_id: providerList[0]?.id ?? "",
												target_model: targetModel || model,
											},
										])
									}
								>
									<PlusIcon />
									Add
								</Button>
							</div>
							{fallbacks.map((fb) => (
								<div
									key={fb.key}
									className="grid gap-2 rounded-md border p-2 sm:grid-cols-[1fr_1fr_auto]"
								>
									<Select
										value={fb.provider_id}
										onValueChange={(v) => {
											const next = v ?? "";
											setFallbacks((prev) =>
												prev.map((row) =>
													row.key === fb.key
														? { ...row, provider_id: next }
														: row,
												),
											);
										}}
									>
										<SelectTrigger className="w-full">
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											{providerList.map((p) => (
												<SelectItem key={p.id} value={p.id}>
													{p.name}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
									<Input
										placeholder="target model"
										value={fb.target_model}
										onChange={(e) => {
											const v = e.target.value;
											setFallbacks((prev) =>
												prev.map((row) =>
													row.key === fb.key
														? { ...row, target_model: v }
														: row,
												),
											);
										}}
									/>
									<Button
										type="button"
										variant="outline"
										size="sm"
										onClick={() =>
											setFallbacks((prev) =>
												prev.filter((row) => row.key !== fb.key),
											)
										}
									>
										Remove
									</Button>
								</div>
							))}
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
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Edit route</SheetTitle>
						<SheetDescription>
							Update model mapping and fallbacks. Publishes a new gateway
							snapshot.
						</SheetDescription>
					</SheetHeader>
					{edit ? (
						<form
							className="flex flex-1 flex-col gap-4 px-4"
							onSubmit={(e) => {
								e.preventDefault();
								if (!editProviderId) return;
								setEditError(null);
								update.mutate(
									{
										routeId: edit.id,
										model: editModel,
										provider_id: editProviderId,
										target_model: editTargetModel || editModel,
										fallbacks: editFallbacks
											.filter((f) => f.provider_id)
											.map(({ provider_id, target_model }) => ({
												provider_id,
												target_model,
											})),
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
							<div className="space-y-2">
								<div className="flex items-center justify-between">
									<Label>Fallbacks</Label>
									<Button
										type="button"
										variant="outline"
										size="sm"
										onClick={() =>
											setEditFallbacks((prev) => [
												...prev,
												{
													key: crypto.randomUUID(),
													provider_id: providerList[0]?.id ?? "",
													target_model: editTargetModel || editModel,
												},
											])
										}
									>
										<PlusIcon />
										Add
									</Button>
								</div>
								{editFallbacks.map((fb) => (
									<div
										key={fb.key}
										className="grid gap-2 rounded-md border p-2 sm:grid-cols-[1fr_1fr_auto]"
									>
										<Select
											value={fb.provider_id}
											onValueChange={(v) => {
												const next = v ?? "";
												setEditFallbacks((prev) =>
													prev.map((row) =>
														row.key === fb.key
															? { ...row, provider_id: next }
															: row,
													),
												);
											}}
										>
											<SelectTrigger className="w-full">
												<SelectValue />
											</SelectTrigger>
											<SelectContent>
												{providerList.map((p) => (
													<SelectItem key={p.id} value={p.id}>
														{p.name}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
										<Input
											placeholder="target model"
											value={fb.target_model}
											onChange={(e) => {
												const v = e.target.value;
												setEditFallbacks((prev) =>
													prev.map((row) =>
														row.key === fb.key
															? { ...row, target_model: v }
															: row,
													),
												);
											}}
										/>
										<Button
											type="button"
											variant="outline"
											size="sm"
											onClick={() =>
												setEditFallbacks((prev) =>
													prev.filter((row) => row.key !== fb.key),
												)
											}
										>
											Remove
										</Button>
									</div>
								))}
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
