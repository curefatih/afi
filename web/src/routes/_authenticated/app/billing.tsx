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
			description="Cost calculation, budgets, alerts, and invoice generation."
			icon={CreditCardIcon}
			context="Billing is not available in this build."
		/>
	);
}
