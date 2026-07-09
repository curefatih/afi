import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/app")({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <div className="flex flex-row h-screen">
      <div className="w-64 h-full border-r">
        <div className="flex-1 p-2 border-b">Sidebar</div>
      </div>
      <div className="flex-1">
        <div className="border-b p-2">Header</div>
      </div>
    </div>
  );
}
