import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/app/settings/limits')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/settings/limits"!</div>
}
