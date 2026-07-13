import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/teams/$teamId')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/_authenticated/teams/$teamId"!</div>
}
