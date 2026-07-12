import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/settings/teams')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/settings/teams"!</div>
}
