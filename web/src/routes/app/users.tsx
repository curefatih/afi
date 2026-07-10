import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/app/users')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/users"!</div>
}
