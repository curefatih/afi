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
import { pageTitle } from "#/lib/page-meta";

export const Route = createFileRoute("/_authenticated/app/hooks")({
	...pageTitle("Hooks"),
	component: RouteComponent,
});

type HookInfo = {
	name: string;
	before_call?: boolean;
	after_call?: boolean;
	before_chat?: boolean;
	after_chat?: boolean;
};

type GatewayHealth = {
	status: string;
	snapshot_version?: number | null;
	provider_types?: string[];
	hooks?: HookInfo[] | string[];
};

function normalizeHooks(raw: GatewayHealth["hooks"]): HookInfo[] {
	if (!raw?.length) return [];
	if (typeof raw[0] === "string") {
		return (raw as string[]).map((name) => ({
			name,
			before_chat: true,
		}));
	}
	return raw as HookInfo[];
}

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

	const hooks = normalizeHooks(health.data?.hooks);
	const providers = health.data?.provider_types ?? [];

	return (
		<PageBody>
			<PageHeader
				title="Hooks"
				description="Lifecycle hooks registered on the gateway."
				info="BeforeCall / AfterCall / BeforeChat / AfterChat hooks registered in cmd/gateway (Go and optional WASM via AFI_WASM_BEFORE_CALL / AFI_WASM_BEFORE_CHAT). gRPC runtimes are not shipped yet."
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
								Register BeforeCall / AfterCall / ChatHook in cmd/gateway (see
								extensions/demohook). For a sample per-tag rate limit, see
								extensions/tagquota (example only — not registered by default).
								Provider SDK adapters live under extensions/ and register via
								Registry.RegisterSDK.
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
										<TableHead>Phases</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{hooks.map((h) => (
										<TableRow key={h.name}>
											<TableCell className="font-medium">{h.name}</TableCell>
											<TableCell>
												<div className="flex flex-wrap gap-1">
													{h.before_call ? (
														<Badge variant="secondary">BeforeCall</Badge>
													) : null}
													{h.after_call ? (
														<Badge variant="outline">AfterCall</Badge>
													) : null}
													{h.before_chat ? (
														<Badge variant="secondary">BeforeChat</Badge>
													) : null}
													{h.after_chat ? (
														<Badge variant="outline">AfterChat</Badge>
													) : null}
												</div>
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
