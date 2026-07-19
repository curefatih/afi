import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Building2Icon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import {
	createOrganizationMutationOptions,
	organizationsQueryOptions,
	toOrganization,
} from "#/api/organization";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Badge } from "#/components/ui/badge";
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
import {
	Sheet,
	SheetContent,
	SheetDescription,
	SheetFooter,
	SheetHeader,
	SheetTitle,
} from "#/components/ui/sheet";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { pageTitle } from "#/lib/page-meta";
import { useActiveOrg, useOrgActions } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/organizations")({
	...pageTitle("Organizations"),
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const { setOrganizations, setActiveOrgById } = useOrgActions();
	const qc = useQueryClient();
	const orgs = useQuery(organizationsQueryOptions());

	const [createOpen, setCreateOpen] = useState(false);
	const [name, setName] = useState("");
	const [error, setError] = useState<string | null>(null);

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
			setName("");
			setCreateOpen(false);
			toast.success("Organization created");
		},
	});

	const list = orgs.data ?? [];

	return (
		<PageBody>
			<PageHeader
				title="Organizations"
				description="Organizations you belong to. Switch the active org here or in the sidebar; invite members on Users."
				actions={
					<Button onClick={() => setCreateOpen(true)}>
						<PlusIcon />
						Create organization
					</Button>
				}
			/>
			<QueryGate
				isPending={orgs.isPending}
				isError={orgs.isError}
				error={orgs.error}
				onRetry={() => orgs.refetch()}
			>
				{list.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<Building2Icon />
							</EmptyMedia>
							<EmptyTitle>No organizations</EmptyTitle>
							<EmptyDescription>
								Create an organization to start managing projects, keys, and
								routing.
							</EmptyDescription>
						</EmptyHeader>
						<EmptyContent>
							<Button onClick={() => setCreateOpen(true)}>
								<PlusIcon />
								Create organization
							</Button>
						</EmptyContent>
					</Empty>
				) : (
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>ID</TableHead>
								<TableHead className="w-48" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{list.map((o) => {
								const isActive = o.id === orgId;
								return (
									<TableRow key={o.id}>
										<TableCell className="font-medium">
											<div className="flex items-center gap-2">
												{o.name}
												{isActive ? (
													<Badge variant="secondary">Active</Badge>
												) : null}
											</div>
										</TableCell>
										<TableCell className="text-muted-foreground font-mono text-xs">
											{o.id}
										</TableCell>
										<TableCell className="space-x-2 text-right">
											{isActive ? (
												<Button
													variant="outline"
													size="sm"
													nativeButton={false}
													render={<Link to="/app/settings/general" />}
												>
													Settings
												</Button>
											) : (
												<Button
													variant="outline"
													size="sm"
													onClick={() => setActiveOrgById(o.id)}
												>
													Switch
												</Button>
											)}
										</TableCell>
									</TableRow>
								);
							})}
						</TableBody>
					</Table>
				)}
			</QueryGate>

			<Sheet open={createOpen} onOpenChange={setCreateOpen}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Create organization</SheetTitle>
						<SheetDescription>
							Creates a new organization and switches you into it.
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4"
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
								},
							);
						}}
					>
						<div className="space-y-1">
							<Label htmlFor="org-name">Name</Label>
							<Input
								id="org-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								required
							/>
						</div>
						{error ? <p className="text-destructive text-xs">{error}</p> : null}
						<SheetFooter>
							<Button
								type="button"
								variant="outline"
								onClick={() => setCreateOpen(false)}
							>
								Cancel
							</Button>
							<Button type="submit" disabled={create.isPending || !name.trim()}>
								{create.isPending ? "Creating…" : "Create"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>
		</PageBody>
	);
}
