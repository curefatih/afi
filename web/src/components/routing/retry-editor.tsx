import type { RetryConfig } from "#/api/routing";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { Switch } from "#/components/ui/switch";
import { cn } from "#/lib/utils";

const DEFAULT_RETRY: RetryConfig = {
	max_attempts: 3,
	backoff: {
		strategy: "fixed",
		base_delay: "100ms",
	},
};

export function defaultRetryConfig(): RetryConfig {
	return {
		max_attempts: DEFAULT_RETRY.max_attempts,
		backoff: { ...DEFAULT_RETRY.backoff },
	};
}

/** Normalize editor state into an API payload (null = disabled). */
export function toRetryPayload(value: RetryConfig | null): RetryConfig | null {
	if (!value) return null;
	const strategy = value.backoff.strategy;
	const base: RetryConfig = {
		max_attempts: Math.max(1, Math.floor(Number(value.max_attempts) || 1)),
		backoff: {
			strategy,
			base_delay: value.backoff.base_delay.trim(),
		},
	};
	if (strategy === "exponential") {
		const maxDelay = value.backoff.max_delay?.trim();
		if (maxDelay) base.backoff.max_delay = maxDelay;
		const mult = value.backoff.multiplier;
		if (mult != null && mult !== 0) base.backoff.multiplier = mult;
	}
	return base;
}

export function validateRetry(value: RetryConfig | null): string | null {
	if (!value) return null;
	if (!Number.isFinite(value.max_attempts) || value.max_attempts < 1) {
		return "Retry max attempts must be at least 1";
	}
	if (!value.backoff.base_delay.trim()) {
		return "Retry base delay is required";
	}
	if (value.backoff.strategy === "exponential") {
		const mult = value.backoff.multiplier;
		if (mult != null && mult !== 0 && mult < 1) {
			return "Retry multiplier must be at least 1";
		}
	}
	return null;
}

type RetryEditorProps = {
	value: RetryConfig | null;
	onChange: (next: RetryConfig | null) => void;
	idPrefix?: string;
	className?: string;
};

export function RetryEditor({
	value,
	onChange,
	idPrefix = "route-retry",
	className,
}: RetryEditorProps) {
	const enabled = value != null;
	const cfg = value ?? DEFAULT_RETRY;
	const exponential = cfg.backoff.strategy === "exponential";

	return (
		<div className={cn("space-y-3", className)}>
			<div className="flex items-center justify-between gap-2">
				<div>
					<Label htmlFor={`${idPrefix}-enabled`}>Retry</Label>
					<p className="text-muted-foreground text-[11px]">
						Same-target retries with backoff before fallbacks (5xx / timeout /
						429).
					</p>
				</div>
				<Switch
					id={`${idPrefix}-enabled`}
					checked={enabled}
					onCheckedChange={(on) => onChange(on ? defaultRetryConfig() : null)}
				/>
			</div>

			{enabled ? (
				<div className="space-y-3 rounded-md border p-3">
					<div className="grid gap-3 sm:grid-cols-2">
						<div className="space-y-1">
							<Label htmlFor={`${idPrefix}-attempts`}>Max attempts</Label>
							<Input
								id={`${idPrefix}-attempts`}
								type="number"
								min={1}
								step={1}
								value={cfg.max_attempts}
								onChange={(e) =>
									onChange({
										...cfg,
										max_attempts: Number(e.target.value) || 1,
									})
								}
							/>
							<p className="text-muted-foreground text-[11px]">
								Includes the first try.
							</p>
						</div>
						<div className="space-y-1">
							<Label>Backoff strategy</Label>
							<Select
								value={cfg.backoff.strategy}
								onValueChange={(v) => {
									const strategy =
										v === "exponential" ? "exponential" : "fixed";
									onChange({
										...cfg,
										backoff: {
											strategy,
											base_delay: cfg.backoff.base_delay || "100ms",
											...(strategy === "exponential"
												? {
														max_delay: cfg.backoff.max_delay || "1s",
														multiplier: cfg.backoff.multiplier || 2,
													}
												: {}),
										},
									});
								}}
							>
								<SelectTrigger className="w-full">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="fixed">Fixed</SelectItem>
									<SelectItem value="exponential">Exponential</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</div>

					<div className="grid gap-3 sm:grid-cols-2">
						<div className="space-y-1">
							<Label htmlFor={`${idPrefix}-base`}>Base delay</Label>
							<Input
								id={`${idPrefix}-base`}
								placeholder="100ms"
								value={cfg.backoff.base_delay}
								onChange={(e) =>
									onChange({
										...cfg,
										backoff: { ...cfg.backoff, base_delay: e.target.value },
									})
								}
							/>
							<p className="text-muted-foreground text-[11px]">
								Go duration (e.g. 100ms, 1s).
							</p>
						</div>
						{exponential ? (
							<div className="space-y-1">
								<Label htmlFor={`${idPrefix}-max`}>Max delay</Label>
								<Input
									id={`${idPrefix}-max`}
									placeholder="1s"
									value={cfg.backoff.max_delay ?? ""}
									onChange={(e) =>
										onChange({
											...cfg,
											backoff: {
												...cfg.backoff,
												max_delay: e.target.value,
											},
										})
									}
								/>
							</div>
						) : null}
					</div>

					{exponential ? (
						<div className="space-y-1 sm:max-w-[50%]">
							<Label htmlFor={`${idPrefix}-mult`}>Multiplier</Label>
							<Input
								id={`${idPrefix}-mult`}
								type="number"
								min={1}
								step={0.1}
								value={cfg.backoff.multiplier ?? 2}
								onChange={(e) =>
									onChange({
										...cfg,
										backoff: {
											...cfg.backoff,
											multiplier: Number(e.target.value) || 2,
										},
									})
								}
							/>
						</div>
					) : null}
				</div>
			) : null}
		</div>
	);
}

export function formatRetrySummary(retry: RetryConfig | null | undefined): string {
	if (!retry) return "—";
	const { max_attempts, backoff } = retry;
	if (backoff.strategy === "exponential") {
		return `${max_attempts}× exp ${backoff.base_delay}`;
	}
	return `${max_attempts}× fixed ${backoff.base_delay}`;
}
