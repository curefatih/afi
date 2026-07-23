import { GripVerticalIcon, PlusIcon } from "lucide-react";
import { Reorder, useDragControls } from "motion/react";
import { useEffect, useState } from "react";
import { flushSync } from "react-dom";
import type { RouteFallback } from "#/api/routing";
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
import { cn } from "#/lib/utils";

export type FallbackRow = RouteFallback & { key: string };

type ProviderOption = {
	id: string;
	name: string;
};

type FallbackListProps = {
	fallbacks: FallbackRow[];
	onChange: (next: FallbackRow[]) => void;
	providers: ProviderOption[];
	defaultTargetModel: string;
	showWeights?: boolean;
	className?: string;
};

function FallbackRowItem({
	fb,
	providers,
	showWeights,
	onUpdate,
	onRemove,
}: {
	fb: FallbackRow;
	providers: ProviderOption[];
	showWeights: boolean;
	onUpdate: (patch: Partial<RouteFallback>) => void;
	onRemove: () => void;
}) {
	const controls = useDragControls();
	const [targetModel, setTargetModel] = useState(fb.target_model);
	const [weight, setWeight] = useState(String(fb.weight ?? 1));

	useEffect(() => {
		setTargetModel(fb.target_model);
	}, [fb.target_model]);

	useEffect(() => {
		setWeight(String(fb.weight ?? 1));
	}, [fb.weight]);

	const commitTargetModel = () => {
		if (targetModel === fb.target_model) return;
		// flushSync so a following submit click sees the committed value
		flushSync(() => {
			onUpdate({ target_model: targetModel });
		});
	};

	const commitWeight = () => {
		const n = Number.parseInt(weight, 10);
		const next = Number.isFinite(n) && n > 0 ? n : 1;
		if (next === (fb.weight ?? 1)) {
			setWeight(String(next));
			return;
		}
		flushSync(() => {
			onUpdate({ weight: next });
		});
		setWeight(String(next));
	};

	return (
		<Reorder.Item
			value={fb.key}
			as="div"
			dragListener={false}
			dragControls={controls}
			className={cn(
				"grid gap-2 rounded-md border bg-background p-2",
				showWeights
					? "sm:grid-cols-[auto_1fr_1fr_5rem_auto]"
					: "sm:grid-cols-[auto_1fr_1fr_auto]",
			)}
		>
			<button
				type="button"
				aria-label="Drag to reorder fallback"
				className="text-muted-foreground hover:text-foreground inline-flex size-8 cursor-grab items-center justify-center touch-none active:cursor-grabbing"
				onPointerDown={(e) => controls.start(e)}
			>
				<GripVerticalIcon className="size-4" />
			</button>
			<Select
				value={fb.provider_id}
				onValueChange={(v) => onUpdate({ provider_id: v ?? "" })}
			>
				<SelectTrigger className="w-full">
					<SelectValue />
				</SelectTrigger>
				<SelectContent>
					{providers.map((p) => (
						<SelectItem key={p.id} value={p.id}>
							{p.name}
						</SelectItem>
					))}
				</SelectContent>
			</Select>
			<Input
				placeholder="target model"
				value={targetModel}
				onChange={(e) => setTargetModel(e.target.value)}
				onBlur={commitTargetModel}
			/>
			{showWeights ? (
				<Input
					type="number"
					min={1}
					step={1}
					aria-label="Fallback weight"
					value={weight}
					onChange={(e) => setWeight(e.target.value)}
					onBlur={commitWeight}
				/>
			) : null}
			<Button type="button" variant="outline" size="sm" onClick={onRemove}>
				Remove
			</Button>
		</Reorder.Item>
	);
}

export function FallbackList({
	fallbacks,
	onChange,
	providers,
	defaultTargetModel,
	showWeights = false,
	className,
}: FallbackListProps) {
	const keys = fallbacks.map((f) => f.key);

	return (
		<div className={cn("space-y-2", className)}>
			<div className="flex items-center justify-between gap-2">
				<div>
					<Label>Fallbacks</Label>
					<p className="text-muted-foreground text-[11px]">
						{showWeights
							? "Weighted pick for the first attempt; list order is failover order. Drag to rearrange."
							: "Tried in list order on 5xx / timeout / 429. Drag to rearrange."}
					</p>
				</div>
				<Button
					type="button"
					variant="outline"
					size="sm"
					onClick={() =>
						onChange([
							...fallbacks,
							{
								key:
									typeof crypto !== "undefined" && crypto.randomUUID
										? crypto.randomUUID()
										: Math.random().toString(36).substring(2, 15),
								provider_id: providers[0]?.id ?? "",
								target_model: defaultTargetModel,
								weight: 1,
							},
						])
					}
				>
					<PlusIcon />
					Add
				</Button>
			</div>
			{fallbacks.length === 0 ? (
				<p className="text-muted-foreground text-xs">No fallbacks yet.</p>
			) : (
				<Reorder.Group
					axis="y"
					values={keys}
					onReorder={(nextKeys) => {
						const byKey = new Map(fallbacks.map((f) => [f.key, f]));
						onChange(
							nextKeys
								.map((k) => byKey.get(k))
								.filter((row): row is FallbackRow => row != null),
						);
					}}
					className="space-y-2"
					as="div"
				>
					{fallbacks.map((fb) => (
						<FallbackRowItem
							key={fb.key}
							fb={fb}
							providers={providers}
							showWeights={showWeights}
							onUpdate={(patch) =>
								onChange(
									fallbacks.map((row) =>
										row.key === fb.key ? { ...row, ...patch } : row,
									),
								)
							}
							onRemove={() =>
								onChange(fallbacks.filter((row) => row.key !== fb.key))
							}
						/>
					))}
				</Reorder.Group>
			)}
		</div>
	);
}
