import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { ClipboardListIcon } from "lucide-react";
import { useMemo } from "react";
import { z } from "zod";
import { AUDIT_EVENT_TYPES, auditQueryOptions } from "#/api/audit";
import { orgMembersQueryOptions } from "#/api/organization";
import {
	DateRangePicker,
	type DateRangePresetId,
	type DateRangeValue,
	defaultDateRange,
	findDateRangePreset,
} from "#/components/date-range-picker";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
import {
	Combobox,
	ComboboxCollection,
	ComboboxContent,
	ComboboxEmpty,
	ComboboxInput,
	ComboboxItem,
	ComboboxList,
} from "#/components/ui/combobox";
import {
	Empty,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";
import { Label } from "#/components/ui/label";
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

const ALL_EVENTS_VALUE = "__all__";

type EventOption = { value: string; label: string };

const EVENT_OPTIONS: EventOption[] = [
	{ value: ALL_EVENTS_VALUE, label: "All events" },
	...AUDIT_EVENT_TYPES.map((e) => ({ value: e.value, label: e.label })),
];

const DATE_RANGE_PRESETS = [
	"today",
	"yesterday",
	"this_week",
	"last_week",
	"this_month",
	"last_month",
	"last_7",
	"last_30",
	"last_90",
] as const satisfies readonly DateRangePresetId[];

const AUDIT_EVENT_VALUES = AUDIT_EVENT_TYPES.map((e) => e.value) as [
	(typeof AUDIT_EVENT_TYPES)[number]["value"],
	...(typeof AUDIT_EVENT_TYPES)[number]["value"][],
];

const auditSearchSchema = z.object({
	name: z.enum(AUDIT_EVENT_VALUES).optional(),
	preset: z.enum(DATE_RANGE_PRESETS).optional(),
	from: z.string().optional(),
	to: z.string().optional(),
});

type AuditSearch = z.infer<typeof auditSearchSchema>;

export const Route = createFileRoute("/_authenticated/app/audit")({
	...pageTitle("Audit"),
	validateSearch: auditSearchSchema,
	component: RouteComponent,
});

function dateRangeFromSearch(search: AuditSearch): DateRangeValue {
	if (search.preset) return defaultDateRange(search.preset);
	if (search.from && search.to) {
		const from = new Date(search.from);
		const to = new Date(search.to);
		if (!Number.isNaN(from.getTime()) && !Number.isNaN(to.getTime())) {
			return { from, to };
		}
	}
	return defaultDateRange("last_30");
}

function formatWhen(iso: string) {
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return iso;
	return d.toLocaleString(undefined, {
		year: "numeric",
		month: "short",
		day: "numeric",
		hour: "2-digit",
		minute: "2-digit",
		second: "2-digit",
	});
}

function actorLabel(r: {
	actor_name?: string;
	actor_email?: string;
	actor_user_id: string;
}) {
	if (r.actor_name && r.actor_email)
		return `${r.actor_name} (${r.actor_email})`;
	if (r.actor_email) return r.actor_email;
	if (r.actor_name) return r.actor_name;
	if (r.actor_user_id) return r.actor_user_id;
	return "—";
}

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const navigate = useNavigate({ from: Route.fullPath });
	const search = Route.useSearch();
	const members = useQuery(orgMembersQueryOptions(orgId));

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const dateRange = useMemo(() => dateRangeFromSearch(search), [search]);
	const eventName = search.name ?? "";
	const selectedEvent = useMemo(
		() =>
			EVENT_OPTIONS.find((o) => o.value === (eventName || ALL_EVENTS_VALUE)) ??
			EVENT_OPTIONS[0],
		[eventName],
	);

	function patchSearch(patch: Partial<AuditSearch>) {
		void navigate({
			search: (prev: AuditSearch) => ({
				...prev,
				...patch,
			}),
			replace: true,
		});
	}

	function setDateRange(range: DateRangeValue) {
		const preset = findDateRangePreset(range);
		if (preset) {
			patchSearch({
				preset,
				from: undefined,
				to: undefined,
			});
			return;
		}
		patchSearch({
			preset: undefined,
			from: range.from.toISOString(),
			to: range.to.toISOString(),
		});
	}

	function setEventType(option: EventOption | null) {
		if (!option || option.value === ALL_EVENTS_VALUE) {
			patchSearch({ name: undefined });
			return;
		}
		patchSearch({ name: option.value as AuditSearch["name"] });
	}

	const filters = useMemo(
		() => ({
			limit: 100,
			from: dateRange.from.toISOString(),
			to: dateRange.to.toISOString(),
			name: eventName || undefined,
		}),
		[dateRange.from, dateRange.to, eventName],
	);

	const audit = useQuery({
		...auditQueryOptions(orgId, filters),
		enabled: !!orgId && isOrgAdmin,
	});

	return (
		<PageBody>
			<PageHeader
				title="Audit"
				description="Org-scoped trail of control-plane mutations for compliance and forensics."
				info="Written on every successful platform domain event (keys, policies, members, credentials, …). Org admins only. Separate from the optional event outbox used for brokers."
			/>

			{!isOrgAdmin ? (
				<Empty className="border min-h-64">
					<EmptyHeader>
						<EmptyMedia variant="icon">
							<ClipboardListIcon />
						</EmptyMedia>
						<EmptyTitle>Admins only</EmptyTitle>
						<EmptyDescription>
							Audit history is available to organization owners and admins.
						</EmptyDescription>
					</EmptyHeader>
				</Empty>
			) : (
				<>
					<div className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
						<DateRangePicker value={dateRange} onChange={setDateRange} />
						<div className="space-y-1">
							<Label className="inline-flex items-center gap-1.5">
								<ClipboardListIcon
									className="size-3.5 text-muted-foreground"
									aria-hidden
								/>
								Event type
							</Label>
							<Combobox
								items={EVENT_OPTIONS}
								value={selectedEvent}
								onValueChange={setEventType}
								itemToStringLabel={(item) => item.label}
								isItemEqualToValue={(a, b) => a.value === b.value}
							>
								<ComboboxInput
									placeholder="Search events…"
									className="w-full"
									showClear={selectedEvent.value !== ALL_EVENTS_VALUE}
								/>
								<ComboboxContent align="start">
									<ComboboxEmpty>No matching events</ComboboxEmpty>
									<ComboboxList>
										<ComboboxCollection>
											{(item) => (
												<ComboboxItem key={item.value} value={item}>
													{item.label}
												</ComboboxItem>
											)}
										</ComboboxCollection>
									</ComboboxList>
								</ComboboxContent>
							</Combobox>
						</div>
					</div>

					<QueryGate
						isPending={audit.isPending || members.isPending}
						isError={audit.isError}
						error={audit.error}
						onRetry={() => audit.refetch()}
					>
						{(audit.data ?? []).length === 0 ? (
							<Empty className="border min-h-64">
								<EmptyHeader>
									<EmptyMedia variant="icon">
										<ClipboardListIcon />
									</EmptyMedia>
									<EmptyTitle>No audit events</EmptyTitle>
									<EmptyDescription>
										Mutations in this org will appear here after they succeed.
									</EmptyDescription>
								</EmptyHeader>
							</Empty>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead className="w-[180px]">When</TableHead>
										<TableHead>Summary</TableHead>
										<TableHead>Event</TableHead>
										<TableHead>Actor</TableHead>
										<TableHead className="font-mono text-xs">
											Resource
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{(audit.data ?? []).map((r) => (
										<TableRow key={r.id}>
											<TableCell className="whitespace-nowrap text-muted-foreground text-sm">
												{formatWhen(r.at)}
											</TableCell>
											<TableCell className="font-medium">{r.summary}</TableCell>
											<TableCell>
												<Badge
													variant="secondary"
													className="font-mono text-xs"
												>
													{r.name}
												</Badge>
											</TableCell>
											<TableCell className="text-sm">{actorLabel(r)}</TableCell>
											<TableCell className="max-w-[200px] truncate font-mono text-xs text-muted-foreground">
												{r.resource_id || "—"}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</QueryGate>
				</>
			)}
		</PageBody>
	);
}
