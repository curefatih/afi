import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Building2Icon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { addOrgMemberMutationOptions } from "#/api/organization";
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
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/settings/general")({
	staticData: {
		getTitle: () => "Organization settings",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const qc = useQueryClient();
	const [email, setEmail] = useState("");
	const [error, setError] = useState<string | null>(null);

	const invite = useMutation({
		...addOrgMemberMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["organizations", orgId, "members"],
			});
			setEmail("");
			toast.success("Member added");
		},
	});

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
				description={`Preferences and membership for ${activeOrg.name}. Switch organizations from the sidebar or Organizations page.`}
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
				<div className="space-y-1">
					<h2 className="text-sm font-medium">Invite member</h2>
					<p className="text-muted-foreground text-sm">
						Add an existing platform user by email. No email is sent — the user
						must already have an account.
					</p>
				</div>
				<form
					className="flex flex-col gap-3 sm:flex-row sm:items-end"
					onSubmit={(e) => {
						e.preventDefault();
						setError(null);
						invite.mutate(
							{ orgId, email },
							{
								onError: (err) =>
									setError(
										err instanceof Error ? err.message : "Invite failed",
									),
							},
						);
					}}
				>
					<div className="min-w-0 flex-1 space-y-1">
						<Label htmlFor="member-email">Email</Label>
						<Input
							id="member-email"
							type="email"
							value={email}
							onChange={(e) => setEmail(e.target.value)}
							required
						/>
					</div>
					<Button type="submit" disabled={invite.isPending || !email.trim()}>
						{invite.isPending ? "Adding…" : "Add member"}
					</Button>
				</form>
				{error ? <p className="text-destructive text-xs">{error}</p> : null}
			</section>

			<section className="space-y-3 rounded-md border p-4">
				<h2 className="text-sm font-medium">Related</h2>
				<p className="text-muted-foreground text-sm">
					Manage roles on Users. Configure usage limits on Quotas.
				</p>
				<div className="flex flex-wrap gap-2">
					<Button
						variant="outline"
						nativeButton={false}
						render={<Link to="/app/users" />}
					>
						View members
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
