import { createFileRoute } from "@tanstack/react-router";
import { Settings2Icon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/settings/general")({
	staticData: {
		getTitle: () => "General",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="General settings"
			description="Organization-level preferences and defaults."
			icon={Settings2Icon}
			context="General settings are not available in this build. Use the organization switcher and account page for current controls."
		/>
	);
}
