import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import {
	addOrgMemberMutationOptions,
	createOrganizationMutationOptions,
	orgMembersQueryOptions,
	organizationsQueryOptions,
	toOrganization,
} from "#/api/organization";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { useActiveOrg, useOrgActions } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/organizations")({
	staticData: {
		getTitle: () => "Organizations",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const { setOrganizations, setActiveOrgById } = useOrgActions();
	const qc = useQueryClient();
	const orgs = useQuery(organizationsQueryOptions());
	const members = useQuery(orgMembersQueryOptions(orgId));

	const create = useMutation({
		...createOrganizationMutationOptions(),
		onSuccess: async (created) => {
			await qc.invalidateQueries({ queryKey: ["organizations"] });
			const list = await qc.fetchQuery(organizationsQueryOptions());
			setOrganizations(
				list.map((o) =>
					toOrganization(
						o,
						o.id === activeOrg?.id ? (activeOrg.teams ?? []) : [],
						o.id === activeOrg?.id ? (activeOrg.projects ?? []) : [],
					),
				),
			);
			setActiveOrgById(created.id);
		},
	});

	const invite = useMutation({
		...addOrgMemberMutationOptions(),
		onSuccess: () =>
			qc.invalidateQueries({ queryKey: ["organizations", orgId, "members"] }),
	});

	const [name, setName] = useState("");
	const [email, setEmail] = useState("");
	const [error, setError] = useState<string | null>(null);

	return (
		<PageBody>
			<PageHeader
				title="Organizations"
				description="Create organizations and invite existing platform users by email (no email send — user must already exist)."
			/>
			<QueryGate
				isPending={orgs.isPending}
				isError={orgs.isError}
				error={orgs.error}
				onRetry={() => orgs.refetch()}
			>
				<div className="grid gap-6 lg:grid-cols-2">
					<div className="space-y-3">
						<h3 className="text-sm font-medium">Your organizations</h3>
						<ul className="divide-y rounded-md border">
							{(orgs.data ?? []).map((o) => (
								<li
									key={o.id}
									className="flex items-center justify-between gap-2 p-3 text-sm"
								>
									<div>
										<div className="font-medium">{o.name}</div>
										<div className="text-muted-foreground text-xs">{o.id}</div>
									</div>
									{o.id === orgId ? (
										<span className="text-muted-foreground text-xs">
											Active
										</span>
									) : (
										<Button
											variant="outline"
											size="sm"
											onClick={() => setActiveOrgById(o.id)}
										>
											Switch
										</Button>
									)}
								</li>
							))}
						</ul>

						<form
							className="space-y-3 rounded-md border p-4"
							onSubmit={(e) => {
								e.preventDefault();
								setError(null);
								create.mutate(
									{ name },
									{
										onError: (err) =>
											setError(
												err instanceof Error ? err.message : "Create failed",
											),
										onSuccess: () => setName(""),
									},
								);
							}}
						>
							<h3 className="text-sm font-medium">Create organization</h3>
							<div className="space-y-1">
								<Label htmlFor="org-name">Name</Label>
								<Input
									id="org-name"
									value={name}
									onChange={(e) => setName(e.target.value)}
									required
								/>
							</div>
							{error ? (
								<p className="text-destructive text-xs">{error}</p>
							) : null}
							<Button type="submit" disabled={create.isPending || !name.trim()}>
								Create
							</Button>
						</form>
					</div>

					<div className="space-y-3">
						<h3 className="text-sm font-medium">
							Members{orgId ? ` · ${activeOrg?.name ?? orgId}` : ""}
						</h3>
						{!orgId ? (
							<p className="text-muted-foreground text-sm">
								Select or create an organization.
							</p>
						) : (
							<>
								<ul className="divide-y rounded-md border">
									{(members.data ?? []).map((m) => (
										<li key={m.user_id} className="p-3 text-sm">
											<div className="font-medium">{m.name}</div>
											<div className="text-muted-foreground text-xs">
												{m.email}
											</div>
										</li>
									))}
									{(members.data ?? []).length === 0 && !members.isPending ? (
										<li className="text-muted-foreground p-3 text-sm">
											No members loaded.
										</li>
									) : null}
								</ul>
								<form
									className="space-y-3 rounded-md border p-4"
									onSubmit={(e) => {
										e.preventDefault();
										setError(null);
										invite.mutate(
											{ orgId, email },
											{
												onError: (err) =>
													setError(
														err instanceof Error
															? err.message
															: "Invite failed",
													),
												onSuccess: () => setEmail(""),
											},
										);
									}}
								>
									<h3 className="text-sm font-medium">Add member by email</h3>
									<div className="space-y-1">
										<Label htmlFor="member-email">Email</Label>
										<Input
											id="member-email"
											type="email"
											value={email}
											onChange={(e) => setEmail(e.target.value)}
											required
										/>
									</div>
									<Button
										type="submit"
										disabled={invite.isPending || !email.trim()}
									>
										Add member
									</Button>
								</form>
							</>
						)}
					</div>
				</div>
			</QueryGate>
		</PageBody>
	);
}
