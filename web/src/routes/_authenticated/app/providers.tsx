import { createFileRoute } from "@tanstack/react-router";
import { PlugIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/providers")({
	staticData: {
		getTitle: () => "Providers",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Providers"
			description="Register LLM providers, credentials, models, and capability metadata."
			icon={PlugIcon}
			context="Provider management APIs are not exposed in this build. Seeded providers still flow into gateway snapshots."
		/>
	);
}
