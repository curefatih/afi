import { createFileRoute, notFound, rootRouteId } from "@tanstack/react-router";

/**
 * Catch-all under /app so unknown paths like /app/does-not-exist
 * trigger the root full-page 404 instead of an empty shell Outlet.
 */
export const Route = createFileRoute("/_authenticated/app/$")({
	beforeLoad: () => {
		throw notFound({ routeId: rootRouteId });
	},
});
