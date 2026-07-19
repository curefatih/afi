import { createFileRoute, Link } from "@tanstack/react-router";
import { Building2Icon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";
import { Button } from "#/components/ui/button";

export const Route = createFileRoute("/_authenticated/app/organizations")({
	staticData: {
		getTitle: () => "Organizations",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Organizations"
			description="Create and administer organizations, membership, and invitations."
			icon={Building2Icon}
			context="Organization switching is available from the sidebar. Create/update APIs are not exposed yet."
			actions={
				<Button render={<Link to="/app/dashboard" />}>Back to overview</Button>
			}
		/>
	);
}
