import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	ActivityIcon,
	BoxIcon,
	CircleDotIcon,
	ClockIcon,
	DollarSignIcon,
	FolderIcon,
	GaugeIcon,
	KeyRoundIcon,
	LayersIcon,
	type LucideIcon,
	ShapesIcon,
	TimerIcon,
	UserIcon,
} from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import {
	Area,
	AreaChart,
	Bar,
	BarChart,
	CartesianGrid,
	ComposedChart,
	Line,
	XAxis,
	YAxis,
} from "recharts";
import { orgKeysQueryOptions } from "#/api/keys";
import { orgProjectsQueryOptions } from "#/api/organization";
import {
	formatUsageKey,
	formatUsageKeyKind,
	formatUsageOwner,
	formatUsageOwnerDetail,
	formatUsageQuantity,
	type UsageFilters,
	type UsageSummaryBucket,
	usageQueryOptions,
	usageSummaryQueryOptions,
} from "#/api/usage";
import {
	DateRangePicker,
	type DateRangeValue,
	defaultDateRange,
} from "#/components/date-range-picker";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
import {
	type ChartConfig,
	ChartContainer,
	ChartTooltip,
	ChartTooltipContent,
} from "#/components/ui/chart";
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
import { pageTitle } from "#/lib/page-meta";
import { cn } from "#/lib/utils";
import { useActiveOrg } from "#/state/organization-state";

const usageByDayConfig = {
	requests: {
		label: "Requests",
		color: "var(--chart-2)",
	},
	trend: {
		label: "Trend",
		color: "oklch(0.72 0.18 55)",
	},
} satisfies ChartConfig;

const costByDayConfig = {
	cost_usd: {
		label: "Cost USD",
		color: "var(--chart-1)",
	},
} satisfies ChartConfig;

const requestsByModalityConfig = {
	requests: {
		label: "Requests",
		color: "var(--chart-2)",
	},
} satisfies ChartConfig;

const costByModelConfig = {
	cost_usd: {
		label: "Cost USD",
		color: "var(--chart-3)",
	},
} satisfies ChartConfig;

const costByKeyConfig = {
	cost_usd: {
		label: "Cost USD",
		color: "var(--chart-4)",
	},
} satisfies ChartConfig;

export const Route = createFileRoute("/_authenticated/app/usage")({
	...pageTitle("Usage"),
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

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const [projectId, setProjectId] = useState("");
	const [apiKeyId, setApiKeyId] = useState("");
	const [modality, setModality] = useState("");
	const [model, setModel] = useState("");
	const [dateRange, setDateRange] = useState<DateRangeValue>(() =>
		defaultDateRange("last_30"),
	);

	const baseFilters: UsageFilters = useMemo(() => {
		const f: UsageFilters = {
			from: dateRange.from.toISOString(),
			to: dateRange.to.toISOString(),
		};
		if (projectId) f.project_id = projectId;
		if (apiKeyId) f.api_key_id = apiKeyId;
		if (modality) f.modality = modality;
		return f;
	}, [projectId, apiKeyId, modality, dateRange]);

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

	const usageByDay = useMemo(
		() => withPaddedUsageDays(byDay.data ?? [], dateRange.from, dateRange.to),
		[byDay.data, dateRange.from, dateRange.to],
	);

	return (
		<PageBody>
			<PageHeader
				title="Usage"
				description="Gateway requests across chat, TTS, STT, and other modalities. Events appear after the usage worker drains the outbox."
			/>

			<div className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
				<DateRangePicker
					className="sm:col-span-2 lg:col-span-1"
					value={dateRange}
					onChange={setDateRange}
				/>
				<div className="space-y-1">
					<Label className="inline-flex items-center gap-1.5">
						<FolderIcon
							className="size-3.5 text-muted-foreground"
							aria-hidden
						/>
						Project
					</Label>
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
					<Label className="inline-flex items-center gap-1.5">
						<KeyRoundIcon
							className="size-3.5 text-muted-foreground"
							aria-hidden
						/>
						API key
					</Label>
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
					<Label className="inline-flex items-center gap-1.5">
						<LayersIcon
							className="size-3.5 text-muted-foreground"
							aria-hidden
						/>
						Modality
					</Label>
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
					<Label className="inline-flex items-center gap-1.5">
						<ShapesIcon
							className="size-3.5 text-muted-foreground"
							aria-hidden
						/>
						Model
					</Label>
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

						<ChartCard title="Usage by day">
							<ChartContainer
								config={usageByDayConfig}
								className="aspect-auto h-[260px] w-full"
							>
								<ComposedChart
									accessibilityLayer
									data={usageByDay}
									margin={{ left: 0, right: 8 }}
									barCategoryGap="18%"
								>
									<CartesianGrid vertical={false} />
									<XAxis
										dataKey="bucket"
										tickLine={false}
										axisLine={false}
										tickMargin={8}
										tick={{ fontSize: 11 }}
										tickFormatter={formatDayTick}
										minTickGap={24}
										padding={{ left: 8, right: 8 }}
									/>
									<YAxis
										tickLine={false}
										axisLine={false}
										width={40}
										tick={{ fontSize: 11 }}
										allowDecimals={false}
									/>
									<ChartTooltip
										content={
											<ChartTooltipContent
												indicator="dot"
												labelFormatter={(_, payload) => {
													const item = payload?.[0]?.payload as
														| { bucket?: string }
														| undefined;
													return item?.bucket
														? formatDayLabel(item.bucket)
														: "";
												}}
											/>
										}
									/>
									<Bar
										dataKey="requests"
										fill="var(--color-requests)"
										fillOpacity={0.55}
										radius={4}
										maxBarSize={40}
									/>
									<Line
										type="monotone"
										dataKey="trend"
										stroke="var(--color-trend)"
										strokeWidth={1.5}
										strokeLinecap="round"
										dot={false}
										connectNulls={false}
										activeDot={{
											r: 4,
											strokeWidth: 2,
											stroke: "var(--background)",
											fill: "var(--color-trend)",
										}}
									/>
								</ComposedChart>
							</ChartContainer>
						</ChartCard>

						<div className="grid gap-6 lg:grid-cols-2">
							<ChartCard title="Cost by day">
								<ChartContainer
									config={costByDayConfig}
									className="aspect-auto h-[220px] w-full"
								>
									<AreaChart
										accessibilityLayer
										data={byDay.data ?? []}
										margin={{ left: 0, right: 8 }}
									>
										<CartesianGrid vertical={false} />
										<XAxis
											dataKey="bucket"
											tickLine={false}
											axisLine={false}
											tickMargin={8}
											tick={{ fontSize: 11 }}
											tickFormatter={formatDayTick}
											minTickGap={24}
										/>
										<YAxis
											tickLine={false}
											axisLine={false}
											width={48}
											tick={{ fontSize: 11 }}
										/>
										<ChartTooltip
											content={
												<ChartTooltipContent
													indicator="line"
													formatter={formatCostTooltip}
													labelFormatter={(_, payload) => {
														const item = payload?.[0]?.payload as
															| { bucket?: string }
															| undefined;
														return item?.bucket
															? formatDayLabel(item.bucket)
															: "";
													}}
												/>
											}
										/>
										<Area
											type="monotone"
											dataKey="cost_usd"
											stroke="var(--color-cost_usd)"
											fill="var(--color-cost_usd)"
											fillOpacity={0.2}
										/>
									</AreaChart>
								</ChartContainer>
							</ChartCard>
							<ChartCard title="Requests by modality">
								<ChartContainer
									config={requestsByModalityConfig}
									className="aspect-auto h-[220px] w-full"
								>
									<BarChart
										accessibilityLayer
										data={byModality.data ?? []}
										margin={{ left: 0, right: 8 }}
									>
										<CartesianGrid vertical={false} />
										<XAxis
											dataKey="label"
											tickLine={false}
											axisLine={false}
											tickMargin={8}
											tick={{ fontSize: 11 }}
										/>
										<YAxis
											tickLine={false}
											axisLine={false}
											width={40}
											tick={{ fontSize: 11 }}
										/>
										<ChartTooltip
											content={<ChartTooltipContent indicator="dot" />}
										/>
										<Bar
											dataKey="requests"
											fill="var(--color-requests)"
											radius={4}
										/>
									</BarChart>
								</ChartContainer>
							</ChartCard>
							<ChartCard title="Cost by model">
								<ChartContainer
									config={costByModelConfig}
									className="aspect-auto h-[220px] w-full"
								>
									<BarChart
										accessibilityLayer
										data={byModel.data ?? []}
										margin={{ left: 0, right: 8 }}
									>
										<CartesianGrid vertical={false} />
										<XAxis
											dataKey="label"
											tickLine={false}
											axisLine={false}
											tickMargin={8}
											tick={{ fontSize: 11 }}
										/>
										<YAxis
											tickLine={false}
											axisLine={false}
											width={48}
											tick={{ fontSize: 11 }}
										/>
										<ChartTooltip
											content={
												<ChartTooltipContent
													indicator="dot"
													formatter={formatCostTooltip}
												/>
											}
										/>
										<Bar
											dataKey="cost_usd"
											fill="var(--color-cost_usd)"
											radius={4}
										/>
									</BarChart>
								</ChartContainer>
							</ChartCard>
							<ChartCard title="Cost by key">
								<ChartContainer
									config={costByKeyConfig}
									className="aspect-auto h-[220px] w-full"
								>
									<BarChart
										accessibilityLayer
										data={byKey.data ?? []}
										margin={{ left: 0, right: 8 }}
									>
										<CartesianGrid vertical={false} />
										<XAxis
											dataKey="label"
											tickLine={false}
											axisLine={false}
											tickMargin={8}
											tick={{ fontSize: 11 }}
										/>
										<YAxis
											tickLine={false}
											axisLine={false}
											width={48}
											tick={{ fontSize: 11 }}
										/>
										<ChartTooltip
											content={
												<ChartTooltipContent
													indicator="dot"
													labelFormatter={(_, payload) => {
														const item = payload?.[0]?.payload as
															| UsageSummaryBucket
															| undefined;
														if (!item) return "";
														const owner =
															item.key_kind === "service_account"
																? "Service account"
																: item.owner_email;
														return owner
															? `${item.label} · ${owner}`
															: item.label;
													}}
													formatter={formatCostTooltip}
												/>
											}
										/>
										<Bar
											dataKey="cost_usd"
											fill="var(--color-cost_usd)"
											radius={4}
										/>
									</BarChart>
								</ChartContainer>
							</ChartCard>
						</div>

						<div className="overflow-x-auto rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<IconHead icon={ClockIcon} label="When" />
										<IconHead icon={LayersIcon} label="Modality" />
										<IconHead icon={UserIcon} label="Owner / scope" />
										<IconHead icon={KeyRoundIcon} label="Key" />
										<IconHead icon={BoxIcon} label="Model" />
										<IconHead icon={CircleDotIcon} label="Status" />
										<IconHead icon={TimerIcon} label="Latency" />
										<IconHead icon={GaugeIcon} label="Usage" />
										<IconHead icon={DollarSignIcon} label="Cost" />
									</TableRow>
								</TableHeader>
								<TableBody>
									{(usage.data ?? []).map((e) => {
										const ownerDetail = formatUsageOwnerDetail(e);
										const keyKind = formatUsageKeyKind(e);
										return (
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
														{ownerDetail ? (
															<span className="text-muted-foreground text-xs">
																{ownerDetail}
															</span>
														) : null}
													</div>
												</TableCell>
												<TableCell>
													<div className="flex flex-col gap-1">
														<span className="font-medium">
															{formatUsageKey(e)}
														</span>
														{keyKind ? (
															<Badge
																variant="outline"
																className="w-fit text-xs font-normal"
															>
																{keyKind}
															</Badge>
														) : null}
													</div>
												</TableCell>
												<TableCell>
													<Badge
														variant="outline"
														className="max-w-48 truncate font-normal"
														title={e.model}
													>
														{e.model}
													</Badge>
												</TableCell>
												<TableCell>
													<StatusBadge status={e.status} />
												</TableCell>
												<TableCell className="tabular-nums">
													{e.latency_ms}ms
												</TableCell>
												<TableCell className="tabular-nums">
													{formatUsageQuantity(e)}
												</TableCell>
												<TableCell className="tabular-nums">
													{e.cost_usd == null
														? "—"
														: `$${e.cost_usd.toFixed(6)}`}
												</TableCell>
											</TableRow>
										);
									})}
								</TableBody>
							</Table>
						</div>
					</div>
				)}
			</QueryGate>
		</PageBody>
	);
}

/**
 * Fill every UTC day in the selected range (empty days pad left/right of real data).
 * Trend is computed only across the span that has real buckets so the line
 * does not stretch through the padding.
 */
function withPaddedUsageDays(
	data: UsageSummaryBucket[],
	from: Date,
	to: Date,
): Array<UsageSummaryBucket & { trend: number | null }> {
	const byBucket = new Map(data.map((d) => [d.bucket, d]));
	const days = eachUTCDay(from, to).map(
		(bucket) => byBucket.get(bucket) ?? emptyDayBucket(bucket),
	);
	if (days.length === 0) return [];

	const dataBuckets = new Set(data.map((d) => d.bucket));
	const coreStart = days.findIndex((d) => dataBuckets.has(d.bucket));
	let coreEnd = -1;
	for (let i = days.length - 1; i >= 0; i--) {
		if (dataBuckets.has(days[i].bucket)) {
			coreEnd = i;
			break;
		}
	}

	// Data outside the selected range: fall back to the data span only.
	if (coreStart < 0 || coreEnd < 0) {
		const fallback = eachUTCDaySpan(data).map(
			(bucket) => byBucket.get(bucket) ?? emptyDayBucket(bucket),
		);
		return withLinearTrend(fallback, "requests").map((d) => ({
			...d,
			trend: d.trend,
		}));
	}

	const core = days.slice(coreStart, coreEnd + 1);
	const trended = withLinearTrend(core, "requests");

	return days.map((d, i) => {
		if (i < coreStart || i > coreEnd) {
			return { ...d, trend: null };
		}
		return { ...d, trend: trended[i - coreStart].trend };
	});
}

function emptyDayBucket(bucket: string): UsageSummaryBucket {
	return {
		bucket,
		label: bucket,
		requests: 0,
		cost_usd: 0,
		prompt_tokens: 0,
		completion_tokens: 0,
	};
}

function eachUTCDay(from: Date, to: Date): string[] {
	const start = Date.UTC(
		from.getUTCFullYear(),
		from.getUTCMonth(),
		from.getUTCDate(),
	);
	const end = Date.UTC(to.getUTCFullYear(), to.getUTCMonth(), to.getUTCDate());
	if (end < start) return [];
	const out: string[] = [];
	for (let t = start; t <= end; t += 86_400_000) {
		out.push(formatUTCDay(t));
	}
	return out;
}

function eachUTCDaySpan(data: UsageSummaryBucket[]): string[] {
	if (data.length === 0) return [];
	const buckets = data.map((d) => d.bucket).sort();
	const start = parseDayBucket(buckets[0]);
	const end = parseDayBucket(buckets[buckets.length - 1]);
	if (!start || !end) return buckets;
	return eachUTCDay(start, end);
}

function formatUTCDay(ms: number): string {
	const d = new Date(ms);
	const y = d.getUTCFullYear();
	const m = String(d.getUTCMonth() + 1).padStart(2, "0");
	const day = String(d.getUTCDate()).padStart(2, "0");
	return `${y}-${m}-${day}`;
}

function withLinearTrend<T extends Record<string, unknown>>(
	data: T[],
	key: keyof T & string,
): Array<T & { trend: number }> {
	const n = data.length;
	if (n === 0) return [];
	if (n === 1) {
		const y = Number(data[0][key]);
		return [{ ...data[0], trend: Number.isFinite(y) ? y : 0 }];
	}

	let sumX = 0;
	let sumY = 0;
	let sumXY = 0;
	let sumXX = 0;
	for (let i = 0; i < n; i++) {
		const y = Number(data[i][key]);
		const value = Number.isFinite(y) ? y : 0;
		sumX += i;
		sumY += value;
		sumXY += i * value;
		sumXX += i * i;
	}
	const denom = n * sumXX - sumX * sumX;
	const slope = denom === 0 ? 0 : (n * sumXY - sumX * sumY) / denom;
	const intercept = (sumY - slope * sumX) / n;

	return data.map((d, i) => ({
		...d,
		trend: Math.round((intercept + slope * i) * 100) / 100,
	}));
}

function formatDayTick(value: string) {
	const d = parseDayBucket(value);
	if (!d) return value;
	return d.toLocaleDateString(undefined, {
		month: "short",
		day: "numeric",
		timeZone: "UTC",
	});
}

function formatDayLabel(value: string) {
	const d = parseDayBucket(value);
	if (!d) return value;
	return d.toLocaleDateString(undefined, {
		weekday: "short",
		month: "short",
		day: "numeric",
		year: "numeric",
		timeZone: "UTC",
	});
}

function parseDayBucket(value: string): Date | null {
	const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
	if (!match) return null;
	const d = new Date(
		Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3])),
	);
	return Number.isNaN(d.getTime()) ? null : d;
}

function formatCostTooltip(
	value: unknown,
	_name: unknown,
	item: { color?: string; payload?: { fill?: string } },
) {
	const amount =
		typeof value === "number"
			? value
			: typeof value === "string"
				? Number(value)
				: Number.NaN;
	const indicatorColor = item.payload?.fill ?? item.color;
	return (
		<>
			<div
				className="h-2.5 w-2.5 shrink-0 rounded-[2px]"
				style={{ backgroundColor: indicatorColor }}
			/>
			<div className="flex flex-1 items-center justify-between leading-none">
				<span className="text-muted-foreground">Cost USD</span>
				<span className="font-mono font-medium text-foreground tabular-nums">
					{Number.isFinite(amount) ? `$${amount.toFixed(6)}` : String(value)}
				</span>
			</div>
		</>
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

function IconHead({ icon: Icon, label }: { icon: LucideIcon; label: string }) {
	return (
		<TableHead>
			<span className="inline-flex items-center gap-1.5">
				<Icon className="text-muted-foreground size-3.5 shrink-0" aria-hidden />
				{label}
			</span>
		</TableHead>
	);
}

function StatusBadge({ status }: { status: string }) {
	const normalized = status || "unknown";
	const isOk = normalized === "ok";
	const isError =
		normalized === "error" ||
		normalized === "upstream_error" ||
		normalized.includes("error");

	return (
		<Badge
			variant={isError ? "destructive" : isOk ? "secondary" : "outline"}
			className={cn(
				"font-normal capitalize",
				isOk &&
					"border-transparent bg-emerald-500/15 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300",
			)}
		>
			{normalized.replaceAll("_", " ")}
		</Badge>
	);
}
