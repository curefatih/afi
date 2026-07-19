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
			description="Vault-backed secret storage and rotation."
			icon={ShieldIcon}
			context="No secret vault in this build. Provider credentials are env var names (api_key_env) set on the gateway process — configure them under Providers."
		/>
	);
}
