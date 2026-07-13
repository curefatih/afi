import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/app/keys")({
  staticData: {
    getTitle: () => "Keys",
  },
  component: RouteComponent,
});

function RouteComponent() {
  return <div>Hello "/app/keys"!</div>;
}
