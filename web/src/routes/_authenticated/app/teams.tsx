import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute("/_authenticated/app/teams")({
  staticData: {
    getTitle: () => "Team",
  },
  component: RouteComponent,
});

function RouteComponent() {
  return <div>Hello "/app/teams"!</div>
}
