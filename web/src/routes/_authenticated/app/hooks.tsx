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
			description="Lifecycle plugins (gRPC / WASM) for request mutation and enrichment."
			icon={PuzzleIcon}
			context="No hooks/plugin runtime UI in this build. Extensibility today is in-process ChatProvider registration — see docs/development/providers.md."
		/>
	);
}
