import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/app/settings/general")({
  staticData: {
    getTitle: () => "Settings",
  },
  component: RouteComponent,
});

function RouteComponent() {
  return <div>Hello "/app/settings/"!</div>;
}
