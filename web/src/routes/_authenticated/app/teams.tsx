import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/teams')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/teams"!</div>
}
