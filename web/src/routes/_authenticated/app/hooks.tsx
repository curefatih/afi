import { createFileRoute } from "@tanstack/react-router";
import { PuzzleIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/hooks")({
	staticData: {
		getTitle: () => "Hooks",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Hooks"
			description="Lifecycle plugins and runtime extensions for request mutation and enrichment."
			icon={PuzzleIcon}
			context="Extension management UI is not available in this build. Extensions register through the gateway runtime."
		/>
	);
}
