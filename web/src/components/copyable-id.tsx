import { CheckIcon, CopyIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "#/components/ui/button";
import { cn } from "#/lib/utils";

type CopyableIdProps = {
	value: string;
	className?: string;
};

export function CopyableId({ value, className }: CopyableIdProps) {
	const [copied, setCopied] = useState(false);

	async function handleCopy() {
		if (!value) return;
		await navigator.clipboard.writeText(value);
		setCopied(true);
		toast.success("Copied to clipboard");
		setTimeout(() => setCopied(false), 1500);
	}

	return (
		<span className="group/copyable inline-flex max-w-full items-center gap-1">
			<span className={cn("font-mono text-xs break-all", className)}>
				{value}
			</span>
			<Button
				type="button"
				variant="ghost"
				size="icon-xs"
				aria-label="Copy ID"
				disabled={!value}
				className="shrink-0 opacity-0 transition-opacity group-hover/copyable:opacity-100 group-focus-within/copyable:opacity-100 max-md:opacity-100"
				onClick={() => void handleCopy()}
			>
				{copied ? <CheckIcon /> : <CopyIcon />}
			</Button>
		</span>
	);
}
