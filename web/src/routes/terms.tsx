import { createFileRoute } from "@tanstack/react-router";
import { FileTextIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";
import { pageTitle } from "#/lib/page-meta";

export const Route = createFileRoute("/terms")({
	...pageTitle("Terms", {
		description: "Legal terms for using AFI.",
	}),
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<div className="mx-auto flex min-h-svh max-w-3xl items-center p-6">
			<ComingSoonPage
				title="Terms of Service"
				description="Legal terms for using AFI."
				icon={FileTextIcon}
				context="Terms content has not been published for this deployment."
			/>
		</div>
	);
}
