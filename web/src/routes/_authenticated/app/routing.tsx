import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { PlusIcon, RouteIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { providersQueryOptions } from "#/api/provider";
import {
	createRouteMutationOptions,
	deleteRouteMutationOptions,
	routesQueryOptions,
	type RouteFallback,
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
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/routing")({
	staticData: {
		getTitle: () => "Routing",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const qc = useQueryClient();
	const routes = useQuery(routesQueryOptions(orgId));
	const providers = useQuery(providersQueryOptions(orgId));
	const create = useMutation({
		...createRouteMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "routes"],
			});
			toast.success("Route created");
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
	const [fallbacks, setFallbacks] = useState<RouteFallback[]>([]);
	const [error, setError] = useState<string | null>(null);

	const providerList = providers.data ?? [];
	const selectedProvider = providerId || providerList[0]?.id || "";
	const providerName = (id: string) =>
		providerList.find((p) => p.id === id)?.name ?? id;

	return (
		<PageBody>
			<PageHeader
				title="Routing"
				description="Map requested model names to providers. Optional fallbacks run on 5xx/timeout/429 before the response body is committed."
			/>
			<QueryGate
				isPending={
					!!orgId && (routes.isLoading || providers.isLoading)
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
				) : (
					<div className="grid gap-6 lg:grid-cols-2">
						<div className="space-y-3">
							<h3 className="text-sm font-medium">Routes</h3>
							{(routes.data ?? []).length === 0 ? (
								<Empty className="border min-h-48">
									<EmptyHeader>
										<EmptyMedia variant="icon">
											<RouteIcon />
										</EmptyMedia>
										<EmptyTitle>No routes</EmptyTitle>
										<EmptyDescription>
											Create a virtual model name clients will request.
										</EmptyDescription>
									</EmptyHeader>
								</Empty>
							) : (
								<Table>
									<TableHeader>
										<TableRow>
											<TableHead>Model</TableHead>
											<TableHead>Target</TableHead>
											<TableHead>Provider</TableHead>
											<TableHead>Fallbacks</TableHead>
											<TableHead className="w-24" />
										</TableRow>
									</TableHeader>
									<TableBody>
										{(routes.data ?? []).map((r) => (
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
														<span className="text-muted-foreground text-xs">
															—
														</span>
													) : (
														<div className="flex flex-wrap gap-1">
															{(r.fallbacks ?? []).map((f, i) => (
																<Badge
																	key={`${f.provider_id}-${i}`}
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
												<TableCell>
													<Button
														variant="outline"
														size="sm"
														disabled={del.isPending}
														onClick={() => del.mutate(r.id)}
													>
														Delete
													</Button>
												</TableCell>
											</TableRow>
										))}
									</TableBody>
								</Table>
							)}
						</div>

						<form
							className="space-y-3 rounded-md border p-4"
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
										fallbacks: fallbacks.filter((f) => f.provider_id),
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
							<h3 className="text-sm font-medium">Add route</h3>
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
								{fallbacks.map((fb, idx) => (
									<div
										key={idx}
										className="grid gap-2 rounded-md border p-2 sm:grid-cols-[1fr_1fr_auto]"
									>
										<Select
											value={fb.provider_id}
											onValueChange={(v) => {
												const next = v ?? "";
												setFallbacks((prev) =>
													prev.map((row, i) =>
														i === idx ? { ...row, provider_id: next } : row,
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
													prev.map((row, i) =>
														i === idx ? { ...row, target_model: v } : row,
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
													prev.filter((_, i) => i !== idx),
												)
											}
										>
											Remove
										</Button>
									</div>
								))}
							</div>
							{error ? (
								<p className="text-destructive text-xs">{error}</p>
							) : null}
							<Button
								type="submit"
								disabled={create.isPending || !orgId || !selectedProvider}
							>
								{create.isPending ? "Creating…" : "Create & publish"}
							</Button>
						</form>
					</div>
				)}
			</QueryGate>
		</PageBody>
	);
}
