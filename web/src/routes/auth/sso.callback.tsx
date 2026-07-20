import {
	createFileRoute,
	Link,
	useNavigate,
	useSearch,
} from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { bootstrapSessionFromToken } from "#/api/auth";
import { pageTitle } from "#/lib/page-meta";
import { cn } from "#/lib/utils";

export const Route = createFileRoute("/auth/sso/callback")({
	...pageTitle("SSO Sign-in", {
		description: "Completing single sign-on.",
	}),
	component: RouteComponent,
	validateSearch: (search: Record<string, unknown>) => ({
		token: typeof search.token === "string" ? search.token : undefined,
		error: typeof search.error === "string" ? search.error : undefined,
		redirect: typeof search.redirect === "string" ? search.redirect : undefined,
	}),
});

function RouteComponent() {
	const navigate = useNavigate();
	const search = useSearch({ from: "/auth/sso/callback" });
	const [message, setMessage] = useState("Completing sign-in…");

	useEffect(() => {
		let cancelled = false;
		async function run() {
			if (search.error) {
				setMessage(search.error);
				toast.error(search.error);
				navigate({ to: "/auth/login" });
				return;
			}
			if (!search.token) {
				setMessage("Missing session token");
				toast.error("SSO login failed");
				navigate({ to: "/auth/login" });
				return;
			}
			try {
				await bootstrapSessionFromToken(search.token);
				if (cancelled) return;
				toast.success("Welcome back");
				navigate({
					to: search.redirect || "/app/dashboard",
				});
			} catch (err) {
				if (cancelled) return;
				const msg = err instanceof Error ? err.message : "SSO login failed";
				setMessage(msg);
				toast.error(msg);
				navigate({ to: "/auth/login" });
			}
		}
		void run();
		return () => {
			cancelled = true;
		};
	}, [navigate, search.error, search.redirect, search.token]);

	return (
		<div className={cn("flex flex-col items-center gap-2 py-12 text-center")}>
			<p className="text-sm text-muted-foreground">{message}</p>
			<Link
				to="/auth/login"
				className="text-sm underline-offset-4 hover:underline"
			>
				Back to login
			</Link>
		</div>
	);
}
