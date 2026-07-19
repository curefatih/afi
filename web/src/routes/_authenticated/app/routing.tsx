import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { providersQueryOptions } from "#/api/provider";
import {
	createRouteMutationOptions,
	deleteRouteMutationOptions,
	routesQueryOptions,
} from "#/api/routing";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
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
		onSuccess: () =>
			qc.invalidateQueries({ queryKey: ["organizations", orgId, "routes"] }),
	});
	const del = useMutation({
		...deleteRouteMutationOptions(),
		onSuccess: () =>
			qc.invalidateQueries({ queryKey: ["organizations", orgId, "routes"] }),
	});

	const [model, setModel] = useState("ping-model");
	const [targetModel, setTargetModel] = useState("gpt-4o-mini");
	const [providerId, setProviderId] = useState("");
	const [error, setError] = useState<string | null>(null);

	const providerList = providers.data ?? [];
	const selectedProvider = providerId || providerList[0]?.id || "";

	return (
		<PageBody>
			<PageHeader
				title="Routing"
				description="Map requested model names to providers. Saves publish a new gateway snapshot."
			/>
			<QueryGate
				isPending={routes.isPending || providers.isPending}
				isError={routes.isError || providers.isError}
				error={routes.error || providers.error}
				onRetry={() => {
					void routes.refetch();
					void providers.refetch();
				}}
			>
				<div className="grid gap-6 lg:grid-cols-2">
					<div className="space-y-3">
						<h3 className="text-sm font-medium">Routes</h3>
						<ul className="divide-y rounded-md border">
							{(routes.data ?? []).map((r) => (
								<li
									key={r.id}
									className="flex items-start justify-between gap-2 p-3 text-sm"
								>
									<div>
										<div className="font-medium">{r.model}</div>
										<div className="text-muted-foreground">
											→ {r.target_model} via {r.provider_id}
										</div>
									</div>
									<Button
										variant="outline"
										size="sm"
										disabled={del.isPending}
										onClick={() => del.mutate(r.id)}
									>
										Delete
									</Button>
								</li>
							))}
							{(routes.data ?? []).length === 0 ? (
								<li className="text-muted-foreground p-3 text-sm">
									No routes yet.
								</li>
							) : null}
						</ul>
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
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="route-provider">Provider</Label>
							<select
								id="route-provider"
								className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
								value={selectedProvider}
								onChange={(e) => setProviderId(e.target.value)}
								required
							>
								{providerList.map((p) => (
									<option key={p.id} value={p.id}>
										{p.name} ({p.id})
									</option>
								))}
							</select>
						</div>
						{error ? (
							<p className="text-destructive text-xs">{error}</p>
						) : null}
						<Button
							type="submit"
							disabled={create.isPending || !orgId || !selectedProvider}
						>
							Create & publish
						</Button>
					</form>
				</div>
			</QueryGate>
		</PageBody>
	);
}
