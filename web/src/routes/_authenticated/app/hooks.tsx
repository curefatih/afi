import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { PuzzleIcon } from "lucide-react";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
import {
	Empty,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { GATEWAY_API_URL } from "#/lib/gateway-base";

export const Route = createFileRoute("/_authenticated/app/hooks")({
	staticData: {
		getTitle: () => "Hooks",
	},
	component: RouteComponent,
});

type GatewayHealth = {
	status: string;
	snapshot_version?: number | null;
	provider_types?: string[];
	hooks?: string[];
};

function RouteComponent() {
	const health = useQuery({
		queryKey: ["gateway", "healthz", "extensions"],
		queryFn: async () => {
			const res = await fetch(`${GATEWAY_API_URL}/healthz`);
			if (!res.ok) {
				throw new Error(`Gateway healthz HTTP ${res.status}`);
			}
			return (await res.json()) as GatewayHealth;
		},
		refetchInterval: 10_000,
	});

	const hooks = health.data?.hooks ?? [];
	const providers = health.data?.provider_types ?? [];

	return (
		<PageBody>
			<PageHeader
				title="Hooks"
				description="In-process BeforeChat hooks registered at gateway startup. gRPC/WASM runtimes are not shipped yet."
			/>
			<QueryGate
				isPending={health.isPending}
				isError={health.isError}
				error={health.error}
				onRetry={() => health.refetch()}
			>
				{hooks.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<PuzzleIcon />
							</EmptyMedia>
							<EmptyTitle>No hooks registered</EmptyTitle>
							<EmptyDescription>
								Register ChatHook implementations in cmd/gateway (see
								extensions/demohook). Provider SDK adapters live under
								extensions/ and register via Registry.RegisterSDK.
							</EmptyDescription>
						</EmptyHeader>
					</Empty>
				) : (
					<div className="space-y-6">
						<div>
							<h2 className="mb-2 text-sm font-medium">Active hooks</h2>
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Name</TableHead>
										<TableHead>Kind</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{hooks.map((name) => (
										<TableRow key={name}>
											<TableCell className="font-medium">{name}</TableCell>
											<TableCell>
												<Badge variant="secondary">BeforeChat</Badge>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
						<div>
							<h2 className="mb-2 text-sm font-medium">Provider types</h2>
							<div className="flex flex-wrap gap-1">
								{providers.map((t) => (
									<Badge key={t} variant="outline">
										{t}
									</Badge>
								))}
							</div>
							<p className="text-muted-foreground mt-2 text-xs">
								From {GATEWAY_API_URL}/healthz — includes built-ins and SDK
								extensions such as echo.
							</p>
						</div>
					</div>
				)}
			</QueryGate>
		</PageBody>
	);
}
