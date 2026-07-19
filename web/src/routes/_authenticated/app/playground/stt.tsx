import { createFileRoute } from "@tanstack/react-router";
import { MicIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/playground/stt")({
	staticData: {
		getTitle: () => "STT",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Speech to text"
			description="Test STT provider capabilities through the gateway."
			icon={MicIcon}
			context="STT playground tooling is not wired in this build."
		/>
	);
}
