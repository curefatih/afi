import { createFileRoute } from "@tanstack/react-router";
import { ShieldIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/secrets")({
	staticData: {
		getTitle: () => "Secrets",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Secrets"
			description="Manage secret references and secret provider integrations."
			icon={ShieldIcon}
			context="Secret store management is not available in this build. Provider credentials currently use environment references."
		/>
	);
}
