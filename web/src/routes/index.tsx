import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/')({ component: Home })

function Home() {
  return (
    <div className="p-8">
      <h1 className="text-4xl font-bold">AFI - AI Gateway</h1>
      <p className="mt-4 text-lg">
        AFI is an AI Gateway that allows you to easily build and deploy AI applications.
      </p>
    </div>
  )
}
