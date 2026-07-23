import { Link, useRouter } from "@tanstack/react-router";
import { ArrowLeftIcon, LayoutDashboardIcon } from "lucide-react";
import { motion } from "motion/react";
import { useEffect } from "react";
import { Button } from "#/components/ui/button";
import { useIsAuthenticated } from "#/state/auth-state";

export function NotFoundPage() {
	const router = useRouter();
	const isAuthenticated = useIsAuthenticated();
	const dashboardTo = isAuthenticated ? "/app/dashboard" : "/auth/login";

	useEffect(() => {
		document.title = "Page not found · AFI";
	}, []);

	return (
		<div className="relative flex min-h-svh flex-col items-center justify-center overflow-hidden bg-muted px-6 py-16">
			<div
				aria-hidden
				className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_50%_0%,oklch(0.92_0_0)_0%,transparent_55%)] dark:bg-[radial-gradient(ellipse_at_50%_0%,oklch(0.28_0_0)_0%,transparent_55%)]"
			/>
			<div
				aria-hidden
				className="pointer-events-none absolute inset-0 opacity-[0.35] [background-image:linear-gradient(to_right,oklch(0.7_0_0/0.12)_1px,transparent_1px),linear-gradient(to_bottom,oklch(0.7_0_0/0.12)_1px,transparent_1px)] [background-size:48px_48px] [mask-image:radial-gradient(ellipse_at_center,black_20%,transparent_70%)] dark:opacity-20"
			/>

			<motion.div
				className="relative z-10 flex w-full max-w-lg flex-col items-center text-center"
				initial={{ opacity: 0, y: 12 }}
				animate={{ opacity: 1, y: 0 }}
				transition={{ duration: 0.45, ease: [0.22, 1, 0.36, 1] }}
			>
				<div className="mb-8 flex items-center gap-2 text-sm font-medium text-muted-foreground">
					<img
						src="/logo.svg"
						alt=""
						width={20}
						height={20}
						className="size-5 rounded-md"
					/>
					AFI
				</div>

				<motion.p
					aria-hidden
					className="select-none text-[clamp(5.5rem,22vw,9.5rem)] leading-none font-semibold tracking-tighter text-foreground/[0.07] dark:text-foreground/[0.1]"
					initial={{ opacity: 0, scale: 0.96 }}
					animate={{ opacity: 1, scale: 1 }}
					transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1], delay: 0.05 }}
				>
					404
				</motion.p>

				<motion.div
					className="mt-2 space-y-3"
					initial={{ opacity: 0, y: 8 }}
					animate={{ opacity: 1, y: 0 }}
					transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1], delay: 0.12 }}
				>
					<h1 className="font-heading text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
						This page doesn’t exist
					</h1>
					<p className="mx-auto max-w-md text-sm leading-relaxed text-muted-foreground sm:text-base">
						The link may be broken, or the page may have moved. Head back to the
						dashboard or return to where you came from.
					</p>
				</motion.div>

				<motion.div
					className="mt-8 flex w-full flex-col items-stretch justify-center gap-2 sm:w-auto sm:flex-row sm:items-center"
					initial={{ opacity: 0, y: 8 }}
					animate={{ opacity: 1, y: 0 }}
					transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1], delay: 0.2 }}
				>
					<Button
						size="lg"
						className="gap-2"
						nativeButton={false}
						render={<Link to={dashboardTo} />}
					>
						<LayoutDashboardIcon data-icon="inline-start" />
						Go to dashboard
					</Button>
					<Button
						size="lg"
						variant="outline"
						className="gap-2"
						onClick={() => {
							if (router.history.canGoBack()) {
								router.history.back();
								return;
							}
							router.history.push(dashboardTo);
						}}
					>
						<ArrowLeftIcon data-icon="inline-start" />
						Go back
					</Button>
				</motion.div>
			</motion.div>
		</div>
	);
}
