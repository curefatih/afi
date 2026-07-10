import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/app/playground')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/playground"!</div>
}
