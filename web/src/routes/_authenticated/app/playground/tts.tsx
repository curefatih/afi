import { createFileRoute } from "@tanstack/react-router";
import { AudioLinesIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/playground/tts")({
	staticData: {
		getTitle: () => "TTS",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Text to speech"
			description="Test TTS provider capabilities through the gateway."
			icon={AudioLinesIcon}
			context="TTS playground tooling is not wired in this build."
		/>
	);
}
