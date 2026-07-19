import { createFileRoute } from "@tanstack/react-router";
import { UsersIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/users")({
	staticData: {
		getTitle: () => "Users",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Users"
			description="Identity administration for platform users, roles, and permissions."
			icon={UsersIcon}
			context="User administration APIs are not available in this build."
		/>
	);
}
