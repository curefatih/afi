import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { GlobeIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import {
	createRegionMutationOptions,
	type Region,
	regionsQueryOptions,
} from "#/api/regions";
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
import { useAuthUser } from "#/state/auth-state";

export const Route = createFileRoute("/_authenticated/app/regions/")({
	...pageTitle("Regions"),
	component: RouteComponent,
});

function RouteComponent() {
	const user = useAuthUser();
	const isAdmin = user?.role === "admin";
	const qc = useQueryClient();
	const q = useQuery(regionsQueryOptions());
	const [open, setOpen] = useState(false);
	const [slug, setSlug] = useState("");
	const [name, setName] = useState("");

	const create = useMutation({
		...createRegionMutationOptions(),
		onSuccess: async (created) => {
			toast.success("Region created");
			setOpen(false);
			setSlug("");
			setName("");
			qc.setQueryData<Region[]>(["regions"], (prev) => {
				const list = prev ?? [];
				if (list.some((r) => r.id === created.id)) return list;
				return [...list, created];
			});
			await qc.invalidateQueries({ queryKey: ["regions"] });
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const regions = q.data ?? [];

	if (!isAdmin) {
		return (
			<>
				<PageHeader title="Regions" />
				<PageBody>
					<p className="text-muted-foreground text-sm">
						Platform admin access is required to manage regions and
						gateway deployments.
					</p>
				</PageBody>
			</>
		);
	}

	return (
		<>
			<PageHeader
				title="Regions"
				description="Hub-and-spoke gateway deployments across regions."
				actions={
					<Button onClick={() => setOpen(true)}>
						<PlusIcon />
						Add region
					</Button>
				}
			/>
			<PageBody>
				<QueryGate
					isPending={q.isPending}
					isError={q.isError}
					error={q.error}
					onRetry={() => void q.refetch()}
				>
					{regions.length === 0 ? (
						<Empty className="border min-h-64">
							<EmptyHeader>
								<EmptyMedia variant="icon">
									<GlobeIcon />
								</EmptyMedia>
								<EmptyTitle>No regions yet</EmptyTitle>
								<EmptyDescription>
									Create a region, then register gateway deployments that
									heartbeat to the control plane.
								</EmptyDescription>
							</EmptyHeader>
							<EmptyContent>
								<Button onClick={() => setOpen(true)}>
									Add region
								</Button>
							</EmptyContent>
						</Empty>
					) : (
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Name</TableHead>
										<TableHead>Slug</TableHead>
										<TableHead>Status</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{regions.map((r) => (
										<TableRow key={r.id}>
											<TableCell>
												<Link
													to="/app/regions/$regionId"
													params={{ regionId: r.id }}
													className="font-medium underline-offset-4 hover:underline"
												>
													{r.name}
												</Link>
											</TableCell>
											<TableCell className="font-mono text-sm">
												{r.slug}
											</TableCell>
											<TableCell>
												<Badge variant="secondary">{r.status}</Badge>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
					)}
				</QueryGate>
			</PageBody>

			<Sheet open={open} onOpenChange={setOpen}>
				<SheetContent className="overflow-y-auto sm:max-w-md">
					<SheetHeader>
						<SheetTitle>Create region</SheetTitle>
						<SheetDescription>
							Slug is stable (e.g. eu-west) and used in ops config.
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!name.trim() || !slug.trim() || create.isPending) return;
							create.mutate({
								name: name.trim(),
								slug: slug.trim(),
							});
						}}
					>
						<div className="grid gap-2">
							<Label htmlFor="region-name">Name</Label>
							<Input
								id="region-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="EU West"
								autoFocus
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="region-slug">Slug</Label>
							<Input
								id="region-slug"
								value={slug}
								onChange={(e) => setSlug(e.target.value)}
								placeholder="eu-west"
							/>
						</div>
						<SheetFooter className="gap-2 px-0 sm:flex-col sm:space-x-0">
							<Button
								type="button"
								variant="outline"
								onClick={() => setOpen(false)}
							>
								Cancel
							</Button>
							<Button
								type="submit"
								disabled={!name.trim() || !slug.trim() || create.isPending}
							>
								{create.isPending ? "Creating…" : "Create"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>
		</>
	);
}
