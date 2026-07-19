import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import {
	createQuotaMutationOptions,
	deleteQuotaMutationOptions,
	quotasQueryOptions,
} from "#/api/quota";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/quotas")({
	staticData: {
		getTitle: () => "Quotas",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const qc = useQueryClient();
	const quotas = useQuery(quotasQueryOptions(orgId));
	const create = useMutation({
		...createQuotaMutationOptions(),
		onSuccess: () =>
			qc.invalidateQueries({ queryKey: ["organizations", orgId, "quotas"] }),
	});
	const del = useMutation({
		...deleteQuotaMutationOptions(),
		onSuccess: () =>
			qc.invalidateQueries({ queryKey: ["organizations", orgId, "quotas"] }),
	});

	const [scopeType, setScopeType] = useState("organization");
	const [scopeID, setScopeID] = useState(orgId);
	const [metric, setMetric] = useState("requests");
	const [limitValue, setLimitValue] = useState("100");
	const [error, setError] = useState<string | null>(null);

	if (scopeType === "organization" && scopeID !== orgId && orgId) {
		// keep org scope_id in sync when org loads
	}

	return (
		<PageBody>
			<PageHeader
				title="Quotas"
				description="Lifetime and token lifetime limits. Changes publish into the gateway snapshot and are enforced before upstream calls."
			/>
			<QueryGate
				isPending={quotas.isPending}
				isError={quotas.isError}
				error={quotas.error}
				onRetry={() => quotas.refetch()}
			>
				<div className="grid gap-6 lg:grid-cols-2">
					<div className="space-y-3">
						<h3 className="text-sm font-medium">Configured quotas</h3>
						<ul className="divide-y rounded-md border">
							{(quotas.data ?? []).map((q) => (
								<li
									key={q.id}
									className="flex items-start justify-between gap-2 p-3 text-sm"
								>
									<div>
										<div className="font-medium">
											{q.metric} ≤ {q.limit_value} ({q.window})
										</div>
										<div className="text-muted-foreground">
											{q.scope_type}: {q.scope_id}
										</div>
									</div>
									<Button
										variant="outline"
										size="sm"
										disabled={del.isPending}
										onClick={() => del.mutate(q.id)}
									>
										Delete
									</Button>
								</li>
							))}
							{(quotas.data ?? []).length === 0 ? (
								<li className="text-muted-foreground p-3 text-sm">
									No quotas — traffic is unlimited until you add one.
								</li>
							) : null}
						</ul>
					</div>
					<form
						className="space-y-3 rounded-md border p-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!orgId) return;
							setError(null);
							const sid =
								scopeType === "organization" ? orgId : scopeID.trim();
							create.mutate(
								{
									orgId,
									scope_type: scopeType,
									scope_id: sid,
									metric,
									limit_value: Number(limitValue),
									window: "total",
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
						<h3 className="text-sm font-medium">Add quota</h3>
						<div className="space-y-1">
							<Label htmlFor="q-scope">Scope</Label>
							<select
								id="q-scope"
								className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
								value={scopeType}
								onChange={(e) => {
									setScopeType(e.target.value);
									if (e.target.value === "organization") setScopeID(orgId);
								}}
							>
								<option value="organization">organization</option>
								<option value="project">project</option>
								<option value="api_key">api_key</option>
							</select>
						</div>
						{scopeType !== "organization" ? (
							<div className="space-y-1">
								<Label htmlFor="q-scope-id">Scope ID</Label>
								<Input
									id="q-scope-id"
									value={scopeID}
									onChange={(e) => setScopeID(e.target.value)}
									placeholder={
										scopeType === "project" ? "proj_local" : "key_local"
									}
									required
								/>
							</div>
						) : null}
						<div className="space-y-1">
							<Label htmlFor="q-metric">Metric</Label>
							<select
								id="q-metric"
								className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
								value={metric}
								onChange={(e) => setMetric(e.target.value)}
							>
								<option value="requests">requests</option>
								<option value="tokens">tokens</option>
							</select>
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
						{error ? (
							<p className="text-destructive text-xs">{error}</p>
						) : null}
						<Button type="submit" disabled={create.isPending || !orgId}>
							Create & publish
						</Button>
					</form>
				</div>
			</QueryGate>
		</PageBody>
	);
}
