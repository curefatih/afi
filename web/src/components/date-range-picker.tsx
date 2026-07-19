import {
	endOfDay,
	endOfMonth,
	endOfWeek,
	format,
	isSameDay,
	startOfDay,
	startOfMonth,
	startOfWeek,
	subDays,
	subMonths,
	subWeeks,
} from "date-fns";
import {
	CalendarClockIcon,
	CalendarDaysIcon,
	CalendarIcon,
	CalendarRangeIcon,
	CalendarSearchIcon,
	ChevronDownIcon,
	HistoryIcon,
	type LucideIcon,
	SunIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import type { DateRange } from "react-day-picker";
import { Button } from "#/components/ui/button";
import { Calendar } from "#/components/ui/calendar";
import { Label } from "#/components/ui/label";
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "#/components/ui/popover";
import { Separator } from "#/components/ui/separator";
import { cn } from "#/lib/utils";

export type DateRangeValue = {
	from: Date;
	to: Date;
};

type PresetId =
	| "today"
	| "yesterday"
	| "this_week"
	| "last_week"
	| "this_month"
	| "last_month"
	| "last_7"
	| "last_30"
	| "last_90";

type Preset = {
	id: PresetId;
	label: string;
	icon: LucideIcon;
	range: () => DateRangeValue;
};

const PRESETS: Preset[] = [
	{
		id: "today",
		label: "Today",
		icon: SunIcon,
		range: () => {
			const now = new Date();
			return { from: startOfDay(now), to: endOfDay(now) };
		},
	},
	{
		id: "yesterday",
		label: "Yesterday",
		icon: HistoryIcon,
		range: () => {
			const d = subDays(new Date(), 1);
			return { from: startOfDay(d), to: endOfDay(d) };
		},
	},
	{
		id: "this_week",
		label: "This week",
		icon: CalendarDaysIcon,
		range: () => {
			const now = new Date();
			return {
				from: startOfWeek(now, { weekStartsOn: 1 }),
				to: endOfDay(now),
			};
		},
	},
	{
		id: "last_week",
		label: "Last week",
		icon: CalendarClockIcon,
		range: () => {
			const last = subWeeks(new Date(), 1);
			return {
				from: startOfWeek(last, { weekStartsOn: 1 }),
				to: endOfWeek(last, { weekStartsOn: 1 }),
			};
		},
	},
	{
		id: "this_month",
		label: "This month",
		icon: CalendarIcon,
		range: () => {
			const now = new Date();
			return { from: startOfMonth(now), to: endOfDay(now) };
		},
	},
	{
		id: "last_month",
		label: "Last month",
		icon: CalendarRangeIcon,
		range: () => {
			const last = subMonths(new Date(), 1);
			return { from: startOfMonth(last), to: endOfMonth(last) };
		},
	},
	{
		id: "last_7",
		label: "Last 7 days",
		icon: CalendarDaysIcon,
		range: () => {
			const now = new Date();
			return { from: startOfDay(subDays(now, 6)), to: endOfDay(now) };
		},
	},
	{
		id: "last_30",
		label: "Last 30 days",
		icon: CalendarRangeIcon,
		range: () => {
			const now = new Date();
			return { from: startOfDay(subDays(now, 29)), to: endOfDay(now) };
		},
	},
	{
		id: "last_90",
		label: "Last 90 days",
		icon: CalendarSearchIcon,
		range: () => {
			const now = new Date();
			return { from: startOfDay(subDays(now, 89)), to: endOfDay(now) };
		},
	},
];

export function defaultDateRange(
	presetId: PresetId = "last_30",
): DateRangeValue {
	const preset =
		PRESETS.find((p) => p.id === presetId) ??
		PRESETS.find((p) => p.id === "last_30");
	if (!preset) {
		const now = new Date();
		return { from: startOfDay(subDays(now, 29)), to: endOfDay(now) };
	}
	return preset.range();
}

function rangesMatch(a: DateRangeValue, b: DateRangeValue): boolean {
	return isSameDay(a.from, b.from) && isSameDay(a.to, b.to);
}

function formatRangeLabel(range: DateRangeValue): string {
	if (isSameDay(range.from, range.to)) {
		return format(range.from, "LLL d, y");
	}
	if (range.from.getFullYear() === range.to.getFullYear()) {
		return `${format(range.from, "LLL d")} – ${format(range.to, "LLL d, y")}`;
	}
	return `${format(range.from, "LLL d, y")} – ${format(range.to, "LLL d, y")}`;
}

type DateRangePickerProps = {
	value: DateRangeValue;
	onChange: (range: DateRangeValue) => void;
	className?: string;
	label?: string;
};

export function DateRangePicker({
	value,
	onChange,
	className,
	label = "Date range",
}: DateRangePickerProps) {
	const [open, setOpen] = useState(false);
	const [draft, setDraft] = useState<DateRange | undefined>(() => ({
		from: value.from,
		to: value.to,
	}));

	const activePresetId = useMemo(() => {
		return PRESETS.find((p) => rangesMatch(value, p.range()))?.id;
	}, [value]);

	const activePreset = PRESETS.find((p) => p.id === activePresetId);
	const TriggerIcon = activePreset?.icon ?? CalendarIcon;

	function applyRange(range: DateRangeValue) {
		onChange(range);
		setDraft({ from: range.from, to: range.to });
		setOpen(false);
	}

	function handleOpenChange(next: boolean) {
		setOpen(next);
		if (next) {
			setDraft({ from: value.from, to: value.to });
		}
	}

	function handleSelect(selected: DateRange | undefined) {
		setDraft(selected);
		if (selected?.from && selected?.to) {
			onChange({
				from: startOfDay(selected.from),
				to: endOfDay(selected.to),
			});
		}
	}

	return (
		<div className={cn("space-y-1", className)}>
			<Label className="inline-flex items-center gap-1.5">
				<CalendarIcon className="size-3.5 text-muted-foreground" aria-hidden />
				{label}
			</Label>
			<Popover open={open} onOpenChange={handleOpenChange}>
				<PopoverTrigger
					render={
						<Button
							variant="outline"
							className="w-full justify-between px-2.5 font-normal"
						/>
					}
				>
					<span className="flex min-w-0 items-center gap-2">
						<TriggerIcon
							className="size-4 shrink-0 text-muted-foreground"
							aria-hidden
						/>
						<span className="truncate">
							{activePreset?.label ?? formatRangeLabel(value)}
						</span>
					</span>
					<ChevronDownIcon
						data-icon="inline-end"
						className="text-muted-foreground"
					/>
				</PopoverTrigger>
				<PopoverContent
					className="w-auto max-w-[calc(100vw-2rem)] gap-0 overflow-hidden p-0"
					align="start"
				>
					<div className="flex flex-col sm:flex-row">
						<div className="flex flex-col gap-0.5 border-b p-2 sm:w-44 sm:border-r sm:border-b-0">
							<div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
								Quick select
							</div>
							{PRESETS.map((preset) => {
								const Icon = preset.icon;
								const active = preset.id === activePresetId;
								return (
									<Button
										key={preset.id}
										variant={active ? "secondary" : "ghost"}
										size="sm"
										className="w-full justify-start gap-2 font-normal"
										onClick={() => applyRange(preset.range())}
									>
										<Icon className="size-3.5 text-muted-foreground" />
										{preset.label}
									</Button>
								);
							})}
						</div>
						<div className="p-2">
							<Calendar
								mode="range"
								numberOfMonths={2}
								resetOnSelect
								defaultMonth={draft?.from ?? value.from}
								selected={draft}
								onSelect={handleSelect}
								disabled={{ after: new Date() }}
							/>
							{draft?.from && !draft.to ? (
								<p className="px-2 pb-1 text-xs text-muted-foreground">
									Select an end date
								</p>
							) : null}
						</div>
					</div>
					<Separator />
					<div className="flex items-center justify-between gap-2 px-3 py-2">
						<span className="inline-flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
							<CalendarRangeIcon className="size-3.5 shrink-0" aria-hidden />
							<span className="truncate">
								{draft?.from
									? draft.to
										? formatRangeLabel({
												from: draft.from,
												to: draft.to,
											})
										: format(draft.from, "LLL d, y")
									: "Pick a range"}
							</span>
						</span>
						{activePresetId ? null : (
							<span className="shrink-0 text-xs text-muted-foreground">
								Custom
							</span>
						)}
					</div>
				</PopoverContent>
			</Popover>
		</div>
	);
}
