import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/hooks')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/_authenticated/app/hooks"!</div>
}
