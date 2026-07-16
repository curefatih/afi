import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/account')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/_authenticated/app/account"!</div>
}
