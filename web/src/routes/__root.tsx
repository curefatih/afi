import { Outlet, createRootRoute } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { TanStackDevtools } from "@tanstack/react-devtools";
import { ThemeProvider } from "@/components/theme-provider";

import "../styles.css";
import { Toaster } from "@/components/ui/sonner";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TooltipProvider } from "#/components/ui/tooltip";

const queryClient = new QueryClient();

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <>
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
