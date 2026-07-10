import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/app/settings/teams')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/settings/teams"!</div>
}
