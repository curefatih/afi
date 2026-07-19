import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/app/settings/teams")({
	beforeLoad: () => {
		throw redirect({ to: "/app/teams" });
	},
	component: () => null,
});
