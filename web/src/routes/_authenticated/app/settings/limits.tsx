import { createFileRoute } from "@tanstack/react-router";

import { LimitsToolbar } from "@/components/limits/toolbar";
import { LimitsStats } from "@/components/limits/stats";
import { PolicyList } from "@/components/limits/policy-list";

export const Route = createFileRoute(
  "/_authenticated/app/settings/limits",
)({
  staticData: {
    getTitle: () => "Limits",
  },
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-8 p-8">
      <LimitsStats />

      <LimitsToolbar />

      <PolicyList />
    </div>
  );
}