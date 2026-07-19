import { createFileRoute } from "@tanstack/react-router";
import { TriangleAlertIcon } from "lucide-react";
import { PolicyList } from "#/components/limits/policy-list";
import { LimitsStats } from "#/components/limits/stats";
import { LimitsToolbar } from "#/components/limits/toolbar";
import { PageBody, PageHeader } from "#/components/page-header";
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { Badge } from "#/components/ui/badge";

export const Route = createFileRoute("/_authenticated/app/quotas")({
	staticData: {
		getTitle: () => "Quotas",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<PageBody>
			<PageHeader
				title="Quotas"
				description="Token, request, budget, and rate limits across organizations, teams, projects, and keys."
				actions={<Badge variant="secondary">UI preview</Badge>}
			/>

			<Alert>
				<TriangleAlertIcon />
				<AlertTitle>Demo UI — not persisted</AlertTitle>
				<AlertDescription>
					This editor is a design preview. Quota policies are not stored by the
					control plane yet.
				</AlertDescription>
			</Alert>

			<div className="flex flex-col gap-8">
				<LimitsStats />
				<LimitsToolbar />
				<PolicyList />
			</div>
		</PageBody>
	);
}
