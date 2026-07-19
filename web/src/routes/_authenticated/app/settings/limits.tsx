import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/app/settings/limits")({
	beforeLoad: () => {
		throw redirect({ to: "/app/quotas" });
	},
	component: () => null,
});
