import {
	useCallback,
	useEffect,
	useId,
	useMemo,
	useRef,
	useState,
} from "react";
import { Button } from "#/components/ui/button";
import { Label } from "#/components/ui/label";
import { Textarea } from "#/components/ui/textarea";
import {
	applyCompletion,
	CEL_EXAMPLES,
	CEL_OPERATORS,
	CEL_VARIABLES,
	type CelSymbol,
	completionContext,
	filterCompletions,
	insertAtCursor,
} from "#/lib/cel-policy-catalog";
import { cn } from "@/lib/utils";

type Props = {
	id?: string;
	value: string;
	onChange: (next: string) => void;
	className?: string;
};

export function CelExpressionEditor({ id, value, onChange, className }: Props) {
	const autoId = useId();
	const fieldId = id ?? autoId;
	const taRef = useRef<HTMLTextAreaElement>(null);
	const listRef = useRef<HTMLDivElement>(null);
	const [cursor, setCursor] = useState(0);
	const [open, setOpen] = useState(false);
	const [active, setActive] = useState(0);

	const ctx = useMemo(() => completionContext(value, cursor), [value, cursor]);
	const suggestions = useMemo(
		() => filterCompletions(ctx.prefix, ctx.afterDot),
		[ctx.prefix, ctx.afterDot],
	);

	const syncCursor = useCallback(() => {
		const el = taRef.current;
		if (!el) return;
		setCursor(el.selectionStart);
	}, []);

	const write = useCallback(
		(next: string, nextCursor: number) => {
			onChange(next);
			requestAnimationFrame(() => {
				const el = taRef.current;
				if (!el) return;
				el.focus();
				el.setSelectionRange(nextCursor, nextCursor);
				setCursor(nextCursor);
			});
		},
		[onChange],
	);

	const pick = useCallback(
		(item: CelSymbol) => {
			const { next, cursor: c } = applyCompletion(value, cursor, item);
			write(next, c);
			setOpen(false);
		},
		[value, cursor, write],
	);

	const insert = useCallback(
		(text: string) => {
			const el = taRef.current;
			const start = el?.selectionStart ?? cursor;
			const end = el?.selectionEnd ?? start;
			const { next, cursor: c } = insertAtCursor(value, start, text, end);
			write(next, c);
			setOpen(false);
		},
		[value, cursor, write],
	);

	useEffect(() => {
		if (!open) return;
		const onDoc = (e: MouseEvent) => {
			if (
				listRef.current?.contains(e.target as Node) ||
				taRef.current?.contains(e.target as Node)
			) {
				return;
			}
			setOpen(false);
		};
		document.addEventListener("mousedown", onDoc);
		return () => document.removeEventListener("mousedown", onDoc);
	}, [open]);

	const requestFields = CEL_VARIABLES.filter((v) => v.group === "Request");
	const keyFields = CEL_VARIABLES.filter((v) => v.group === "Key");

	return (
		<div className={cn("space-y-3", className)}>
			<div className="rounded-lg border bg-muted/30 p-3 text-xs leading-relaxed text-muted-foreground">
				<p className="font-medium text-foreground text-sm mb-1">
					How policies work
				</p>
				<p>
					Write a <span className="text-foreground">boolean</span> CEL
					expression. Every enabled policy must evaluate to{" "}
					<code className="text-foreground">true</code> or the gateway returns{" "}
					<strong className="text-foreground">403</strong>. Type to
					autocomplete, or click a variable / example below.
				</p>
			</div>

			<div className="space-y-1.5">
				<div className="flex items-center justify-between gap-2">
					<Label htmlFor={fieldId}>Expression</Label>
					<span className="text-[11px] text-muted-foreground">
						Tab / Enter to accept · Esc to dismiss · Ctrl+Space
					</span>
				</div>
				<div className="relative">
					<Textarea
						ref={taRef}
						id={fieldId}
						value={value}
						spellCheck={false}
						rows={5}
						required
						aria-autocomplete="list"
						aria-expanded={open}
						aria-controls={`${fieldId}-suggestions`}
						className="font-mono text-xs leading-5 min-h-28"
						placeholder='request.model != "blocked-model"'
						onChange={(e) => {
							const next = e.target.value;
							const pos = e.target.selectionStart;
							onChange(next);
							setCursor(pos);
							const { prefix } = completionContext(next, pos);
							setActive(0);
							setOpen(prefix.length > 0);
						}}
						onClick={syncCursor}
						onKeyUp={syncCursor}
						onSelect={syncCursor}
						onFocus={() => {
							syncCursor();
						}}
						onKeyDown={(e) => {
							if (e.key === " " && (e.ctrlKey || e.metaKey)) {
								e.preventDefault();
								syncCursor();
								setActive(0);
								setOpen(true);
								return;
							}
							if (!open || suggestions.length === 0) return;
							if (e.key === "ArrowDown") {
								e.preventDefault();
								setActive((i) => (i + 1) % suggestions.length);
							} else if (e.key === "ArrowUp") {
								e.preventDefault();
								setActive(
									(i) => (i - 1 + suggestions.length) % suggestions.length,
								);
							} else if (e.key === "Enter" || e.key === "Tab") {
								e.preventDefault();
								const item = suggestions[active];
								if (item) pick(item);
							} else if (e.key === "Escape") {
								e.preventDefault();
								setOpen(false);
							}
						}}
					/>
					{open && suggestions.length > 0 ? (
						<div
							ref={listRef}
							id={`${fieldId}-suggestions`}
							role="listbox"
							className="absolute z-20 mt-1 max-h-56 w-full overflow-auto rounded-lg border bg-popover p-1 shadow-md"
						>
							{suggestions.map((s, i) => (
								<button
									key={s.label + s.insert}
									type="button"
									role="option"
									aria-selected={i === active}
									className={cn(
										"flex w-full flex-col items-start gap-0.5 rounded-md px-2 py-1.5 text-left",
										i === active ? "bg-accent" : "hover:bg-muted/80",
									)}
									onMouseEnter={() => setActive(i)}
									onMouseDown={(e) => {
										e.preventDefault();
										pick(s);
									}}
								>
									<span className="font-mono text-xs font-medium">
										{s.label}
									</span>
									<span className="text-[11px] text-muted-foreground line-clamp-1">
										{s.detail}
									</span>
								</button>
							))}
						</div>
					) : null}
				</div>
			</div>

			<div className="space-y-2">
				<p className="text-xs font-medium text-foreground">Variables</p>
				<div className="space-y-2 rounded-lg border p-2.5">
					<p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wide">
						request
					</p>
					<div className="flex flex-wrap gap-1.5">
						{requestFields.map((v) => (
							<Button
								key={v.label}
								type="button"
								variant="outline"
								size="xs"
								className="font-mono"
								title={v.detail}
								onClick={() => insert(v.insert)}
							>
								{v.label}
							</Button>
						))}
					</div>
					<p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wide pt-1">
						key
					</p>
					<div className="flex flex-wrap gap-1.5">
						{keyFields.map((v) => (
							<Button
								key={v.label}
								type="button"
								variant="outline"
								size="xs"
								className="font-mono"
								title={v.detail}
								onClick={() => insert(v.insert)}
							>
								{v.label}
							</Button>
						))}
					</div>
					<p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wide pt-1">
						operators
					</p>
					<div className="flex flex-wrap gap-1.5">
						{CEL_OPERATORS.map((op) => (
							<Button
								key={op.label}
								type="button"
								variant="secondary"
								size="xs"
								className="font-mono"
								title={op.detail}
								onClick={() => insert(op.insert)}
							>
								{op.label}
							</Button>
						))}
					</div>
				</div>
			</div>

			<div className="space-y-2">
				<p className="text-xs font-medium text-foreground">Examples</p>
				<ul className="space-y-1.5">
					{CEL_EXAMPLES.map((ex) => (
						<li key={ex.title}>
							<button
								type="button"
								className="w-full rounded-lg border px-3 py-2 text-left hover:bg-muted/50 transition-colors"
								onClick={() => write(ex.expression, ex.expression.length)}
							>
								<div className="flex items-baseline justify-between gap-2">
									<span className="text-sm font-medium">{ex.title}</span>
									<span className="text-[11px] text-muted-foreground shrink-0">
										Use
									</span>
								</div>
								<p className="text-[11px] text-muted-foreground mt-0.5">
									{ex.description}
								</p>
								<code className="mt-1 block font-mono text-[11px] text-foreground/90 truncate">
									{ex.expression}
								</code>
							</button>
						</li>
					))}
				</ul>
			</div>
		</div>
	);
}
