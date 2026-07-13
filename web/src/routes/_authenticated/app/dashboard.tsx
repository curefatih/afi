import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/app/dashboard")({
  staticData: {
    getTitle: () => "Dashboard",
  },
  component: RouteComponent,
});

function RouteComponent() {
  return <div>Hello "/app/dashboard"!</div>;
}
