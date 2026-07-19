import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { orgKeysQueryOptions } from "#/api/keys";
import { orgMembersQueryOptions } from "#/api/organization";
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
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { useAuthUser } from "#/state/auth-state";
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
	const user = useAuthUser();
	const qc = useQueryClient();
	const quotas = useQuery(quotasQueryOptions(orgId));
	const members = useQuery(orgMembersQueryOptions(orgId));
	const keys = useQuery(orgKeysQueryOptions(orgId));

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

	const effectiveScopeID =
		scopeType === "organization" ? orgId : scopeID;

	return (
		<PageBody>
			<PageHeader
				title="Quotas"
				description="Lifetime and token lifetime limits. Most specific wins: api key → user → project → organization."
			/>
			<QueryGate
				isPending={quotas.isPending || members.isPending}
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
											{q.scope_type}: {labels(q.scope_type, q.scope_id)}
										</div>
										<div className="text-muted-foreground font-mono text-xs">
											{q.scope_id}
										</div>
									</div>
									{isOrgAdmin ? (
										<Button
											variant="outline"
											size="sm"
											disabled={del.isPending}
											onClick={() => del.mutate(q.id)}
										>
											Delete
										</Button>
									) : null}
								</li>
							))}
							{(quotas.data ?? []).length === 0 ? (
								<li className="text-muted-foreground p-3 text-sm">
									No quotas — traffic is unlimited until you add one.
								</li>
							) : null}
						</ul>
					</div>

					{isOrgAdmin ? (
						<form
							className="space-y-3 rounded-md border p-4"
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
										window: "total",
									},
									{
										onError: (err) =>
											setError(
												err instanceof Error ? err.message : "Create failed",
											),
										onSuccess: () => setError(null),
									},
								);
							}}
						>
							<h3 className="text-sm font-medium">Add quota</h3>
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
							<Button
								type="submit"
								disabled={
									create.isPending || !orgId || !effectiveScopeID
								}
							>
								Create & publish
							</Button>
						</form>
					) : (
						<p className="text-muted-foreground rounded-md border p-4 text-sm">
							Only organization owners and admins can create or delete quotas.
						</p>
					)}
				</div>
			</QueryGate>
		</PageBody>
	);
}
