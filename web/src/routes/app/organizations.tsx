import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/app/organizations')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/organizations"!</div>
}
