import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/app/projects')({
  staticData: {
    getTitle: () => "Projects",
  },
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/app/projects"!</div>
}
