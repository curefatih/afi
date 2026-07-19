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
		<div className="flex min-h-svh flex-col items-center justify-center gap-6 bg-muted p-6 md:p-10">
			<div className="flex w-full max-w-sm flex-col gap-6">
				<div className="flex items-center gap-2 self-center font-medium">
					<img
						src="/logo.svg"
						alt=""
						width={24}
						height={24}
						className="size-6 rounded-md"
					/>
					AFI
				</div>
				<Outlet />
			</div>
		</div>
	);
}
