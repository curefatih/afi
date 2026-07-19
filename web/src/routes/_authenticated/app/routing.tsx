import { createFileRoute } from "@tanstack/react-router";
import { RouteIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/routing")({
	staticData: {
		getTitle: () => "Routing",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Routing"
			description="Define model selection, failover, and cost/latency-aware routing policies."
			icon={RouteIcon}
			context="Routing policy management is not available in the platform UI yet."
		/>
	);
}
