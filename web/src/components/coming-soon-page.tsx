import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { PageBody, PageHeader } from "#/components/page-header";
import { Badge } from "#/components/ui/badge";
import {
	Empty,
	EmptyContent,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";

type ComingSoonPageProps = {
	title: string;
	description: string;
	icon: LucideIcon;
	context?: string;
	actions?: ReactNode;
};

export function ComingSoonPage({
	title,
	description,
	icon: Icon,
	context = "This control-plane capability is not available in the current build.",
	actions,
}: ComingSoonPageProps) {
	return (
		<PageBody>
			<PageHeader
				title={title}
				description={description}
				actions={<Badge variant="secondary">Coming soon</Badge>}
			/>
			<Empty className="border min-h-72">
				<EmptyHeader>
					<EmptyMedia variant="icon">
						<Icon />
					</EmptyMedia>
					<EmptyTitle>{title}</EmptyTitle>
					<EmptyDescription>{context}</EmptyDescription>
				</EmptyHeader>
				{actions ? <EmptyContent>{actions}</EmptyContent> : null}
			</Empty>
		</PageBody>
	);
}
