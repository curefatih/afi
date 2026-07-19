import { createFileRoute, Link } from "@tanstack/react-router";
import { Building2Icon } from "lucide-react";
import { PageBody, PageHeader } from "#/components/page-header";
import { Button } from "#/components/ui/button";
import {
	Empty,
	EmptyContent,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/settings/general")({
	staticData: {
		getTitle: () => "Organization settings",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();

	if (!activeOrg) {
		return (
			<PageBody>
				<PageHeader
					title="Organization settings"
					description="Settings for the active organization."
				/>
				<Empty className="border min-h-64">
					<EmptyHeader>
						<EmptyMedia variant="icon">
							<Building2Icon />
						</EmptyMedia>
						<EmptyTitle>No active organization</EmptyTitle>
						<EmptyDescription>
							Create or switch to an organization first.
						</EmptyDescription>
					</EmptyHeader>
					<EmptyContent>
						<Button
							nativeButton={false}
							render={<Link to="/app/organizations" />}
						>
							Go to Organizations
						</Button>
					</EmptyContent>
				</Empty>
			</PageBody>
		);
	}

	return (
		<PageBody>
			<PageHeader
				title="Organization settings"
				description={`Preferences for ${activeOrg.name}. Switch organizations from the sidebar or Organizations page.`}
			/>

			<section className="space-y-3 rounded-md border p-4">
				<h2 className="text-sm font-medium">Active organization</h2>
				<dl className="grid gap-2 text-sm sm:grid-cols-2">
					<div>
						<dt className="text-muted-foreground text-xs">Name</dt>
						<dd className="font-medium">{activeOrg.name}</dd>
					</div>
					<div>
						<dt className="text-muted-foreground text-xs">ID</dt>
						<dd className="font-mono text-xs">{activeOrg.id}</dd>
					</div>
				</dl>
			</section>

			<section className="space-y-3 rounded-md border p-4">
				<h2 className="text-sm font-medium">Related</h2>
				<p className="text-muted-foreground text-sm">
					Invite members and manage roles on Users. Configure usage limits on
					Quotas.
				</p>
				<div className="flex flex-wrap gap-2">
					<Button
						variant="outline"
						nativeButton={false}
						render={<Link to="/app/users" />}
					>
						Manage members
					</Button>
					<Button
						variant="outline"
						nativeButton={false}
						render={<Link to="/app/quotas" />}
					>
						Manage quotas
					</Button>
				</div>
			</section>
		</PageBody>
	);
}
