import { GripVerticalIcon } from "lucide-react";
import { Reorder, useDragControls } from "motion/react";
import {
	type PolicyThen,
	policyActions,
	type RequestPolicy,
} from "#/api/policies";
import { Button } from "#/components/ui/button";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { cn } from "#/lib/utils";

/** Assign descending priorities from top → bottom (higher runs first). */
export function prioritiesForOrder(count: number): number[] {
	return Array.from({ length: count }, (_, i) => (count - i) * 10);
}

function formatThenSteps(actions: PolicyThen[]): string {
	return actions.map((a) => a.type || "deny").join(" > ");
}

function ThenColumn({ policy }: { policy: RequestPolicy }) {
	const label = formatThenSteps(policyActions(policy));

	return (
		<TableCell className="max-w-[20rem] min-w-[10rem]">
			<span className="block truncate font-mono text-xs" title={label}>
				{label}
			</span>
		</TableCell>
	);
}

type SortablePolicyTableProps = {
	policies: RequestPolicy[];
	canEdit: boolean;
	disabled?: boolean;
	onReorder: (next: RequestPolicy[]) => void;
	onEdit: (policy: RequestPolicy) => void;
	onDelete: (policyId: string) => void;
	deletePending?: boolean;
};

function PolicyRow({
	policy,
	canEdit,
	disabled,
	onEdit,
	onDelete,
	deletePending,
}: {
	policy: RequestPolicy;
	canEdit: boolean;
	disabled?: boolean;
	onEdit: (policy: RequestPolicy) => void;
	onDelete: (policyId: string) => void;
	deletePending?: boolean;
}) {
	const controls = useDragControls();

	return (
		<Reorder.Item
			as="tr"
			value={policy.id}
			dragListener={false}
			dragControls={controls}
			className={cn(
				"border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted",
				disabled && "opacity-60",
			)}
		>
			{canEdit ? (
				<TableCell className="w-10 pr-0">
					<button
						type="button"
						aria-label={`Drag to reorder ${policy.name}`}
						disabled={disabled}
						className="text-muted-foreground hover:text-foreground inline-flex size-8 cursor-grab items-center justify-center touch-none disabled:cursor-not-allowed disabled:opacity-50 active:cursor-grabbing"
						onPointerDown={(e) => {
							if (disabled) return;
							controls.start(e);
						}}
					>
						<GripVerticalIcon className="size-4" />
					</button>
				</TableCell>
			) : null}
			<TableCell className="font-medium">{policy.name}</TableCell>
			<TableCell>{policy.priority}</TableCell>
			<TableCell>{policy.enabled ? "yes" : "no"}</TableCell>
			<ThenColumn policy={policy} />
			<TableCell className="font-mono text-xs max-w-md truncate">
				{policy.expression}
			</TableCell>
			{canEdit ? (
				<TableCell className="space-x-2">
					<Button
						variant="outline"
						size="sm"
						disabled={disabled}
						onClick={() => onEdit(policy)}
					>
						Edit
					</Button>
					<Button
						variant="outline"
						size="sm"
						disabled={disabled || deletePending}
						onClick={() => onDelete(policy.id)}
					>
						Delete
					</Button>
				</TableCell>
			) : null}
		</Reorder.Item>
	);
}

export function SortablePolicyTable({
	policies,
	canEdit,
	disabled,
	onReorder,
	onEdit,
	onDelete,
	deletePending,
}: SortablePolicyTableProps) {
	const ids = policies.map((p) => p.id);

	return (
		<div className="space-y-2">
			{canEdit ? (
				<p className="text-muted-foreground text-xs">
					Drag rows to set evaluation order. Higher in the list runs first
					(higher priority).
				</p>
			) : null}
			<Table>
				<TableHeader>
					<TableRow>
						{canEdit ? <TableHead className="w-10" /> : null}
						<TableHead>Name</TableHead>
						<TableHead>Priority</TableHead>
						<TableHead>Enabled</TableHead>
						<TableHead>Then</TableHead>
						<TableHead>When</TableHead>
						{canEdit ? <TableHead className="w-40" /> : null}
					</TableRow>
				</TableHeader>
				{canEdit ? (
					<Reorder.Group
						as="tbody"
						axis="y"
						values={ids}
						onReorder={(nextIds) => {
							if (disabled) return;
							const byId = new Map(policies.map((p) => [p.id, p]));
							const next = nextIds
								.map((id) => byId.get(id))
								.filter((p): p is RequestPolicy => p != null);
							const priorities = prioritiesForOrder(next.length);
							onReorder(
								next.map((p, i) => ({
									...p,
									priority: priorities[i],
								})),
							);
						}}
						className={cn(
							"[&_tr]:border-b [&_tr:last-child]:border-0",
							disabled && "pointer-events-none",
						)}
					>
						{policies.map((p) => (
							<PolicyRow
								key={p.id}
								policy={p}
								canEdit={canEdit}
								disabled={disabled}
								onEdit={onEdit}
								onDelete={onDelete}
								deletePending={deletePending}
							/>
						))}
					</Reorder.Group>
				) : (
					<TableBody>
						{policies.map((p) => (
							<TableRow key={p.id}>
								<TableCell className="font-medium">{p.name}</TableCell>
								<TableCell>{p.priority}</TableCell>
								<TableCell>{p.enabled ? "yes" : "no"}</TableCell>
								<ThenColumn policy={p} />
								<TableCell className="font-mono text-xs max-w-md truncate">
									{p.expression}
								</TableCell>
							</TableRow>
						))}
					</TableBody>
				)}
			</Table>
		</div>
	);
}
