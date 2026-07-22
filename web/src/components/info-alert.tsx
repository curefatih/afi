import { InfoIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Alert, AlertDescription } from "#/components/ui/alert";
import { cn } from "#/lib/utils";

export function InfoAlert({
	children,
	className,
}: {
	children: ReactNode;
	className?: string;
}) {
	return (
		<Alert role="status" className={cn("mb-3", className)}>
			<InfoIcon />
			<AlertDescription>{children}</AlertDescription>
		</Alert>
	);
}
