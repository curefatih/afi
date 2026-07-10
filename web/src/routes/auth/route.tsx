import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/auth")({
  beforeLoad: ({ location }) => {
    if (location.pathname === "/auth") {
      throw redirect({
        to: "/auth/login",
      });
    }
  },
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <div className="flex flex-row h-screen">
      <div className="w-1/2 h-full border-r border-gray-200 bg-gray-100">
      </div>
      <div className="flex-1">
        <div className="w-full h-full bg-white flex items-center justify-center">
          <Outlet />
        </div>
      </div>
    </div>
  );
}
