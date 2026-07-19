import { createFileRoute } from "@tanstack/react-router";
import { ShieldIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";
import { pageTitle } from "#/lib/page-meta";

export const Route = createFileRoute("/privacy")({
	...pageTitle("Privacy", {
		description: "How AFI handles platform data.",
	}),
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<div className="mx-auto flex min-h-svh max-w-3xl items-center p-6">
			<ComingSoonPage
				title="Privacy Policy"
				description="How AFI handles platform data."
				icon={ShieldIcon}
				context="Privacy content has not been published for this deployment."
			/>
		</div>
	);
}
