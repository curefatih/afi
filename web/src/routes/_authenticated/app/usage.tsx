import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { usageQueryOptions } from "#/api/usage";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/usage")({
	staticData: {
		getTitle: () => "Usage",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const usage = useQuery(usageQueryOptions(orgId));

	return (
		<PageBody>
			<PageHeader
				title="Usage"
				description="Recent gateway chat completions for this organization."
			/>
			<QueryGate
				isPending={usage.isPending}
				isError={usage.isError}
				error={usage.error}
				onRetry={() => usage.refetch()}
			>
				<div className="overflow-x-auto rounded-md border">
					<table className="w-full text-left text-sm">
						<thead className="bg-muted/40 text-muted-foreground">
							<tr>
								<th className="p-2 font-medium">When</th>
								<th className="p-2 font-medium">Model</th>
								<th className="p-2 font-medium">Status</th>
								<th className="p-2 font-medium">Latency</th>
								<th className="p-2 font-medium">Tokens</th>
								<th className="p-2 font-medium">Cost</th>
							</tr>
						</thead>
						<tbody>
							{(usage.data ?? []).map((e) => (
								<tr key={e.id} className="border-t">
									<td className="p-2 whitespace-nowrap">
										{new Date(e.created_at).toLocaleString()}
									</td>
									<td className="p-2">{e.model}</td>
									<td className="p-2">{e.status}</td>
									<td className="p-2 tabular-nums">{e.latency_ms}ms</td>
									<td className="p-2 tabular-nums">
										{e.prompt_tokens}/{e.completion_tokens}
									</td>
									<td className="p-2 tabular-nums">
										{e.cost_usd == null
											? "—"
											: `$${e.cost_usd.toFixed(6)}`}
									</td>
								</tr>
							))}
							{(usage.data ?? []).length === 0 ? (
								<tr>
									<td
										colSpan={6}
										className="text-muted-foreground p-4 text-center"
									>
										No usage events yet. Send a chat through the gateway.
									</td>
								</tr>
							) : null}
						</tbody>
					</table>
				</div>
			</QueryGate>
		</PageBody>
	);
}
