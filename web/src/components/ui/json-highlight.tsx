import { cn } from "@/lib/utils";

type JsonHighlightProps = {
	value: string;
	className?: string;
};

export function JsonHighlight({ value, className }: JsonHighlightProps) {
	let data: unknown;
	try {
		data = JSON.parse(value);
	} catch {
		return (
			<pre
				className={cn(
					"bg-background max-h-[32rem] overflow-auto rounded-lg border p-3 text-xs whitespace-pre-wrap",
					className,
				)}
			>
				{value}
			</pre>
		);
	}

	return (
		<pre
			className={cn(
				"bg-background max-h-[32rem] overflow-auto rounded-lg border p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap break-words",
				className,
			)}
		>
			<JsonValue data={data} />
			{"\n"}
		</pre>
	);
}

function JsonValue({ data, depth = 0 }: { data: unknown; depth?: number }) {
	if (data === null) {
		return <span className="text-violet-700 dark:text-violet-300">null</span>;
	}
	if (typeof data === "boolean") {
		return (
			<span className="text-violet-700 dark:text-violet-300">
				{String(data)}
			</span>
		);
	}
	if (typeof data === "number") {
		return (
			<span className="text-amber-700 dark:text-amber-300">{String(data)}</span>
		);
	}
	if (typeof data === "string") {
		return (
			<span className="text-emerald-700 dark:text-emerald-300">
				{JSON.stringify(data)}
			</span>
		);
	}
	if (Array.isArray(data)) {
		if (data.length === 0) return <>{"[]"}</>;
		const pad = "  ".repeat(depth + 1);
		const close = "  ".repeat(depth);
		return (
			<>
				{"[\n"}
				{data.map((item, i) => (
					<span key={i}>
						{pad}
						<JsonValue data={item} depth={depth + 1} />
						{i < data.length - 1 ? "," : ""}
						{"\n"}
					</span>
				))}
				{close}
				{"]"}
			</>
		);
	}
	if (typeof data === "object") {
		const entries = Object.entries(data as Record<string, unknown>);
		if (entries.length === 0) return <>{"{}"}</>;
		const pad = "  ".repeat(depth + 1);
		const close = "  ".repeat(depth);
		return (
			<>
				{"{\n"}
				{entries.map(([key, val], i) => (
					<span key={key}>
						{pad}
						<span className="text-sky-700 dark:text-sky-300">
							{JSON.stringify(key)}
						</span>
						{": "}
						<JsonValue data={val} depth={depth + 1} />
						{i < entries.length - 1 ? "," : ""}
						{"\n"}
					</span>
				))}
				{close}
				{"}"}
			</>
		);
	}
	return <>{String(data)}</>;
}
