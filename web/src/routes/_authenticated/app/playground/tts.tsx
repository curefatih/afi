import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/playground/tts')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/_authenticated/app/playground/tts"!</div>
}
