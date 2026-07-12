import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/organizations')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/organizations"!</div>
}
