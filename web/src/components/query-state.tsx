import { AlertCircleIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { Button } from "#/components/ui/button";
import { Skeleton } from "#/components/ui/skeleton";

export function PageSkeleton({ rows = 4 }: { rows?: number }) {
	return (
		<div className="space-y-3">
			{Array.from({ length: rows }, (_, i) => `skeleton-${i}`).map((key) => (
				<Skeleton key={key} className="h-16 w-full" />
			))}
		</div>
	);
}

export function QueryError({
	message,
	onRetry,
}: {
	message?: string;
	onRetry?: () => void;
}) {
	return (
		<Alert variant="destructive">
			<AlertCircleIcon />
			<AlertTitle>Something went wrong</AlertTitle>
			<AlertDescription className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<span>{message || "Failed to load data."}</span>
				{onRetry ? (
					<Button variant="outline" size="sm" onClick={onRetry}>
						Retry
					</Button>
				) : null}
			</AlertDescription>
		</Alert>
	);
}

export function QueryGate({
	isPending,
	isError,
	error,
	onRetry,
	children,
	skeleton,
}: {
	isPending: boolean;
	isError: boolean;
	error?: Error | null;
	onRetry?: () => void;
	children: ReactNode;
	skeleton?: ReactNode;
}) {
	if (isPending) return <>{skeleton ?? <PageSkeleton />}</>;
	if (isError) return <QueryError message={error?.message} onRetry={onRetry} />;
	return <>{children}</>;
}
