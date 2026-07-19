import { createFileRoute } from "@tanstack/react-router";
import { CreditCardIcon } from "lucide-react";
import { ComingSoonPage } from "#/components/coming-soon-page";

export const Route = createFileRoute("/_authenticated/app/billing")({
	staticData: {
		getTitle: () => "Billing",
	},
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<ComingSoonPage
			title="Billing"
			description="Invoices, budgets, and payment integrations."
			icon={CreditCardIcon}
			context="No billing or invoice APIs in this build. Usage and cost_usd appear on the Usage page when the worker drains the outbox; quotas cover hard limits."
		/>
	);
}
