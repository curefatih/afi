import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/settings/limits')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/settings/limits"!</div>
}
