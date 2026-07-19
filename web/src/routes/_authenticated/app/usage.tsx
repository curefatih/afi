import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { ActivityIcon } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import {
	Area,
	AreaChart,
	Bar,
	BarChart,
	CartesianGrid,
	ResponsiveContainer,
	Tooltip,
	XAxis,
	YAxis,
} from "recharts";
import { orgKeysQueryOptions } from "#/api/keys";
import { orgProjectsQueryOptions } from "#/api/organization";
import {
	formatUsageOwner,
	formatUsageQuantity,
	type UsageFilters,
	usageQueryOptions,
	usageSummaryQueryOptions,
} from "#/api/usage";
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

export const Route = createFileRoute("/_authenticated/app/usage")({
	staticData: {
		getTitle: () => "Usage",
	},
	component: RouteComponent,
});

const MODALITIES = [
	"chat",
	"messages",
	"tts",
	"stt",
	"embedding",
	"image",
	"video",
];

function rangeFrom(days: number): string {
	const d = new Date();
	d.setUTCDate(d.getUTCDate() - days);
	return d.toISOString();
}

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const [projectId, setProjectId] = useState("");
	const [apiKeyId, setApiKeyId] = useState("");
	const [modality, setModality] = useState("");
	const [model, setModel] = useState("");
	const [rangeDays, setRangeDays] = useState("30");

	const baseFilters: UsageFilters = useMemo(() => {
		const f: UsageFilters = { from: rangeFrom(Number(rangeDays) || 30) };
		if (projectId) f.project_id = projectId;
		if (apiKeyId) f.api_key_id = apiKeyId;
		if (modality) f.modality = modality;
		return f;
	}, [projectId, apiKeyId, modality, rangeDays]);

	const filters: UsageFilters = useMemo(() => {
		const f = { ...baseFilters };
		if (model) f.model = model;
		return f;
	}, [baseFilters, model]);

	const projects = useQuery(orgProjectsQueryOptions(orgId));
	const keys = useQuery(orgKeysQueryOptions(orgId));
	const usage = useQuery(usageQueryOptions(orgId, filters));
	const byDay = useQuery(usageSummaryQueryOptions(orgId, "day", filters));
	const byModality = useQuery(
		usageSummaryQueryOptions(orgId, "modality", filters),
	);
	const byModel = useQuery(usageSummaryQueryOptions(orgId, "model", filters));
	const modelOptions = useQuery(
		usageSummaryQueryOptions(orgId, "model", baseFilters),
	);
	const byKey = useQuery(usageSummaryQueryOptions(orgId, "key", filters));

	const pending =
		usage.isPending ||
		byDay.isPending ||
		byModality.isPending ||
		byModel.isPending ||
		byKey.isPending;
	const error =
		usage.error ||
		byDay.error ||
		byModality.error ||
		byModel.error ||
		byKey.error;
	const isError =
		usage.isError ||
		byDay.isError ||
		byModality.isError ||
		byModel.isError ||
		byKey.isError;

	const totals = useMemo(() => {
		const days = byDay.data ?? [];
		return days.reduce(
			(acc, b) => {
				acc.requests += b.requests;
				acc.cost += b.cost_usd;
				acc.prompt += b.prompt_tokens;
				acc.completion += b.completion_tokens;
				return acc;
			},
			{ requests: 0, cost: 0, prompt: 0, completion: 0 },
		);
	}, [byDay.data]);

	const models = useMemo(() => {
		const set = new Set<string>();
		for (const b of modelOptions.data ?? []) {
			if (b.bucket) set.add(b.bucket);
		}
		return [...set].sort();
	}, [modelOptions.data]);

	const hasAny = (byDay.data?.length ?? 0) > 0 || (usage.data?.length ?? 0) > 0;

	return (
		<PageBody>
			<PageHeader
				title="Usage"
				description="Gateway requests across chat, TTS, STT, and other modalities. Events appear after the usage worker drains the outbox."
			/>

			<div className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
				<div className="space-y-1">
					<Label>Range</Label>
					<Select
						value={rangeDays}
						onValueChange={(v) => setRangeDays(v ?? "30")}
					>
						<SelectTrigger className="w-full">
							<SelectValue />
						</SelectTrigger>
						<SelectContent>
							<SelectItem value="7">Last 7 days</SelectItem>
							<SelectItem value="30">Last 30 days</SelectItem>
							<SelectItem value="90">Last 90 days</SelectItem>
						</SelectContent>
					</Select>
				</div>
				<div className="space-y-1">
					<Label>Project</Label>
					<Select
						value={projectId || "__all__"}
						onValueChange={(v) =>
							setProjectId(v === "__all__" ? "" : (v ?? ""))
						}
					>
						<SelectTrigger className="w-full">
							<SelectValue placeholder="All projects" />
						</SelectTrigger>
						<SelectContent>
							<SelectItem value="__all__">All projects</SelectItem>
							{(projects.data ?? []).map((p) => (
								<SelectItem key={p.id} value={p.id}>
									{p.name}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</div>
				<div className="space-y-1">
					<Label>API key</Label>
					<Select
						value={apiKeyId || "__all__"}
						onValueChange={(v) => setApiKeyId(v === "__all__" ? "" : (v ?? ""))}
					>
						<SelectTrigger className="w-full">
							<SelectValue placeholder="All keys" />
						</SelectTrigger>
						<SelectContent>
							<SelectItem value="__all__">All keys</SelectItem>
							{(keys.data ?? []).map((k) => (
								<SelectItem key={k.id} value={k.id}>
									{k.name}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</div>
				<div className="space-y-1">
					<Label>Modality</Label>
					<Select
						value={modality || "__all__"}
						onValueChange={(v) => setModality(v === "__all__" ? "" : (v ?? ""))}
					>
						<SelectTrigger className="w-full">
							<SelectValue placeholder="All modalities" />
						</SelectTrigger>
						<SelectContent>
							<SelectItem value="__all__">All modalities</SelectItem>
							{MODALITIES.map((m) => (
								<SelectItem key={m} value={m}>
									{m}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</div>
				<div className="space-y-1">
					<Label>Model</Label>
					<Select
						value={model || "__all__"}
						onValueChange={(v) => setModel(v === "__all__" ? "" : (v ?? ""))}
					>
						<SelectTrigger className="w-full">
							<SelectValue placeholder="All models" />
						</SelectTrigger>
						<SelectContent>
							<SelectItem value="__all__">All models</SelectItem>
							{models.map((m) => (
								<SelectItem key={m} value={m}>
									{m}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</div>
			</div>

			<QueryGate
				isPending={pending}
				isError={isError}
				error={error}
				onRetry={() => {
					void usage.refetch();
					void byDay.refetch();
					void byModality.refetch();
					void byModel.refetch();
					void byKey.refetch();
				}}
			>
				{!hasAny ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<ActivityIcon />
							</EmptyMedia>
							<EmptyTitle>No usage yet</EmptyTitle>
							<EmptyDescription>
								Send chat, TTS, or STT through the gateway, and keep{" "}
								<code className="text-xs">make run-worker</code> running so
								events land here.
							</EmptyDescription>
						</EmptyHeader>
					</Empty>
				) : (
					<div className="space-y-6">
						<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
							<Stat label="Requests" value={String(totals.requests)} />
							<Stat
								label="Cost"
								value={totals.cost > 0 ? `$${totals.cost.toFixed(4)}` : "—"}
							/>
							<Stat
								label="Prompt tokens"
								value={totals.prompt > 0 ? String(totals.prompt) : "—"}
							/>
							<Stat
								label="Completion tokens"
								value={totals.completion > 0 ? String(totals.completion) : "—"}
							/>
						</div>

						<div className="grid gap-6 lg:grid-cols-2">
							<ChartCard title="Cost by day">
								<ResponsiveContainer width="100%" height={220}>
									<AreaChart data={byDay.data ?? []}>
										<CartesianGrid
											strokeDasharray="3 3"
											className="stroke-border"
										/>
										<XAxis dataKey="bucket" tick={{ fontSize: 11 }} />
										<YAxis tick={{ fontSize: 11 }} width={48} />
										<Tooltip />
										<Area
											type="monotone"
											dataKey="cost_usd"
											name="Cost USD"
											stroke="var(--chart-1)"
											fill="var(--chart-2)"
										/>
									</AreaChart>
								</ResponsiveContainer>
							</ChartCard>
							<ChartCard title="Requests by modality">
								<ResponsiveContainer width="100%" height={220}>
									<BarChart data={byModality.data ?? []}>
										<CartesianGrid
											strokeDasharray="3 3"
											className="stroke-border"
										/>
										<XAxis dataKey="label" tick={{ fontSize: 11 }} />
										<YAxis tick={{ fontSize: 11 }} width={40} />
										<Tooltip />
										<Bar
											dataKey="requests"
											name="Requests"
											fill="var(--chart-2)"
										/>
									</BarChart>
								</ResponsiveContainer>
							</ChartCard>
							<ChartCard title="Cost by model">
								<ResponsiveContainer width="100%" height={220}>
									<BarChart data={byModel.data ?? []}>
										<CartesianGrid
											strokeDasharray="3 3"
											className="stroke-border"
										/>
										<XAxis dataKey="label" tick={{ fontSize: 11 }} />
										<YAxis tick={{ fontSize: 11 }} width={48} />
										<Tooltip />
										<Bar
											dataKey="cost_usd"
											name="Cost USD"
											fill="var(--chart-3)"
										/>
									</BarChart>
								</ResponsiveContainer>
							</ChartCard>
							<ChartCard title="Cost by key">
								<ResponsiveContainer width="100%" height={220}>
									<BarChart data={byKey.data ?? []}>
										<CartesianGrid
											strokeDasharray="3 3"
											className="stroke-border"
										/>
										<XAxis dataKey="label" tick={{ fontSize: 11 }} />
										<YAxis tick={{ fontSize: 11 }} width={48} />
										<Tooltip
											formatter={(value, _name, item) => {
												const p = item?.payload as
													| { owner_email?: string; key_kind?: string }
													| undefined;
												const owner =
													p?.key_kind === "service_account"
														? "Service account"
														: p?.owner_email || "";
												return [
													typeof value === "number"
														? `$${value.toFixed(6)}`
														: String(value),
													owner ? `Cost (${owner})` : "Cost USD",
												];
											}}
										/>
										<Bar
											dataKey="cost_usd"
											name="Cost USD"
											fill="var(--chart-4)"
										/>
									</BarChart>
								</ResponsiveContainer>
							</ChartCard>
						</div>

						<div className="overflow-x-auto rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>When</TableHead>
										<TableHead>Modality</TableHead>
										<TableHead>Owner</TableHead>
										<TableHead>Key</TableHead>
										<TableHead>Model</TableHead>
										<TableHead>Status</TableHead>
										<TableHead>Latency</TableHead>
										<TableHead>Usage</TableHead>
										<TableHead>Cost</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{(usage.data ?? []).map((e) => (
										<TableRow key={e.id}>
											<TableCell className="whitespace-nowrap">
												{new Date(e.created_at).toLocaleString()}
											</TableCell>
											<TableCell>
												<Badge variant="secondary">
													{e.modality || "chat"}
												</Badge>
											</TableCell>
											<TableCell>
												<div className="flex flex-col">
													<span>{formatUsageOwner(e)}</span>
													{e.owner_email && e.key_kind !== "service_account" ? (
														<span className="text-muted-foreground text-xs">
															{e.owner_email}
														</span>
													) : null}
												</div>
											</TableCell>
											<TableCell>{e.key_name || e.api_key_id || "—"}</TableCell>
											<TableCell>{e.model}</TableCell>
											<TableCell>{e.status}</TableCell>
											<TableCell className="tabular-nums">
												{e.latency_ms}ms
											</TableCell>
											<TableCell className="tabular-nums">
												{formatUsageQuantity(e)}
											</TableCell>
											<TableCell className="tabular-nums">
												{e.cost_usd == null ? "—" : `$${e.cost_usd.toFixed(6)}`}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
					</div>
				)}
			</QueryGate>
		</PageBody>
	);
}

function Stat({ label, value }: { label: string; value: string }) {
	return (
		<div className="rounded-md border px-3 py-2">
			<div className="text-muted-foreground text-xs">{label}</div>
			<div className="text-lg font-medium tabular-nums">{value}</div>
		</div>
	);
}

function ChartCard({
	title,
	children,
}: {
	title: string;
	children: ReactNode;
}) {
	return (
		<div className="rounded-md border p-3">
			<div className="mb-2 text-sm font-medium">{title}</div>
			{children}
		</div>
	);
}
