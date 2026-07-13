import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/playground')({
  staticData: {
    getTitle: () => "Playground",
  },
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/playground"!</div>
}
