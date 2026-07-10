import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/app/teams')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/teams"!</div>
}
