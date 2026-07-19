import type { ReactNode } from "react";
import { cn } from "#/lib/utils";

type PageHeaderProps = {
	title: string;
	description?: string;
	actions?: ReactNode;
	className?: string;
};

export function PageHeader({
	title,
	description,
	actions,
	className,
}: PageHeaderProps) {
	return (
		<div
			className={cn(
				"flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between",
				className,
			)}
		>
			<div className="min-w-0 space-y-1">
				<h1 className="scroll-m-20 text-2xl font-semibold tracking-tight">
					{title}
				</h1>
				{description ? (
					<p className="text-sm text-muted-foreground text-pretty max-w-2xl">
						{description}
					</p>
				) : null}
			</div>
			{actions ? (
				<div className="flex shrink-0 flex-wrap items-center gap-2">
					{actions}
				</div>
			) : null}
		</div>
	);
}

export function PageBody({
	children,
	className,
}: {
	children: ReactNode;
	className?: string;
}) {
	return (
		<div className={cn("flex flex-1 flex-col gap-6", className)}>
			{children}
		</div>
	);
}
