import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/playground/stt')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/_authenticated/app/playground/stt"!</div>
}
