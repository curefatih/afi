import { TanStackDevtools } from "@tanstack/react-devtools";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRootRoute, HeadContent, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { NotFoundPage } from "#/components/not-found-page";
import { TooltipProvider } from "#/components/ui/tooltip";
import { defaultAppDescription } from "#/lib/page-meta";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/sonner";

import "../styles.css";

const queryClient = new QueryClient();

export const Route = createRootRoute({
	head: () => ({
		meta: [
			{ title: "AFI" },
			{ name: "description", content: defaultAppDescription },
			{ name: "application-name", content: "AFI" },
			{ name: "theme-color", content: "#0b0f14" },
			{ property: "og:type", content: "website" },
			{ property: "og:site_name", content: "AFI" },
			{ property: "og:title", content: "AFI" },
			{ property: "og:description", content: defaultAppDescription },
			{ name: "twitter:card", content: "summary" },
			{ name: "twitter:title", content: "AFI" },
			{ name: "twitter:description", content: defaultAppDescription },
		],
		links: [
			{ rel: "icon", href: "/favicon.ico", sizes: "any" },
			{
				rel: "icon",
				href: "/favicon-32.png",
				type: "image/png",
				sizes: "32x32",
			},
			{
				rel: "icon",
				href: "/favicon-16.png",
				type: "image/png",
				sizes: "16x16",
			},
			{ rel: "apple-touch-icon", href: "/apple-touch-icon.png" },
			{ rel: "manifest", href: "/manifest.json" },
		],
	}),
	notFoundComponent: NotFoundPage,
	component: RootComponent,
});

function RootComponent() {
	return (
		<>
			<HeadContent />
			<ThemeProvider defaultTheme="system" storageKey="theme">
				<QueryClientProvider client={queryClient}>
					<TooltipProvider>
						<Outlet />
					</TooltipProvider>
					<Toaster position="bottom-right" />
				</QueryClientProvider>
			</ThemeProvider>
			<TanStackDevtools
				config={{
					position: "bottom-right",
				}}
				plugins={[
					{
						name: "TanStack Router",
						render: <TanStackRouterDevtoolsPanel />,
					},
				]}
			/>
		</>
	);
}
