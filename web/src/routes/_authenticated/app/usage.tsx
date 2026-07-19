import { createFileRoute } from "@tanstack/react-router";
import { BarChart3Icon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/usage")({
	staticData: {
		getTitle: () => "Usage",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Usage"
			description="Token accounting, latency metrics, and external user tag attribution."
			icon={BarChart3Icon}
			context="Usage dashboards will consume control-plane analytics APIs that are not ready yet."
		/>
	);
}
