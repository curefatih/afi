import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/app/keys')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/keys"!</div>
}
