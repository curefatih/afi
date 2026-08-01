import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { PlusIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { organizationsQueryOptions } from "#/api/organization";
import {
	bindAllOrgsMutationOptions,
	bindOrgMutationOptions,
	deleteOverlayMutationOptions,
	deploymentsQueryOptions,
	putOverlayMutationOptions,
	regionMembershipsQueryOptions,
	regionQueryOptions,
	registerDeploymentMutationOptions,
	rotateJoinTokenMutationOptions,
	unbindOrgMutationOptions,
	updateRegionMutationOptions,
} from "#/api/regions";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { RegionOverlaySheet } from "#/components/regions/overlay-sheet";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
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

export const Route = createFileRoute("/_authenticated/app/regions/$regionId")({
	...pageTitle("Region"),
	component: RouteComponent,
});

function statusVariant(status: string) {
	if (status === "healthy" || status === "active") return "default" as const;
	if (status === "stale" || status === "disabled")
		return "destructive" as const;
	return "secondary" as const;
}

function RouteComponent() {
	const { regionId } = Route.useParams();
	const user = useAuthUser();
	const isAdmin = user?.role === "admin";
	const qc = useQueryClient();
	const regionQ = useQuery(regionQueryOptions(regionId));
	const depsQ = useQuery(deploymentsQueryOptions(regionId));
	const memQ = useQuery(regionMembershipsQueryOptions(regionId));
	const orgsQ = useQuery(organizationsQueryOptions());
	const [open, setOpen] = useState(false);
	const [bindOpen, setBindOpen] = useState(false);
	const [overlayOrgId, setOverlayOrgId] = useState<string | null>(null);
	const [name, setName] = useState("");
	const [baseURL, setBaseURL] = useState("");
	const [joinToken, setJoinToken] = useState<string | null>(null);
	const [bindOrgId, setBindOrgId] = useState("");

	const update = useMutation({
		...updateRegionMutationOptions(),
		onSuccess: async () => {
			toast.success("Region updated");
			await qc.invalidateQueries({ queryKey: ["regions", regionId] });
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const register = useMutation({
		...registerDeploymentMutationOptions(),
		onSuccess: async (out) => {
			toast.success("Deployment registered");
			setJoinToken(out.join_token);
			setOpen(false);
			setName("");
			setBaseURL("");
			await qc.invalidateQueries({
				queryKey: ["regions", regionId, "deployments"],
			});
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const rotate = useMutation({
		...rotateJoinTokenMutationOptions(),
		onSuccess: (out) => {
			setJoinToken(out.join_token);
			toast.success("Join token rotated");
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const bind = useMutation({
		...bindOrgMutationOptions(),
		onSuccess: async () => {
			toast.success("Organization bound");
			setBindOpen(false);
			setBindOrgId("");
			await qc.invalidateQueries({
				queryKey: ["regions", regionId, "organizations"],
			});
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const bindAll = useMutation({
		...bindAllOrgsMutationOptions(),
		onSuccess: async (out) => {
			toast.success(`Bound ${out.bound} organization(s)`);
			await qc.invalidateQueries({
				queryKey: ["regions", regionId, "organizations"],
			});
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const unbind = useMutation({
		...unbindOrgMutationOptions(),
		onSuccess: async () => {
			toast.success("Organization unbound");
			await qc.invalidateQueries({
				queryKey: ["regions", regionId, "organizations"],
			});
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const putOverlay = useMutation({
		...putOverlayMutationOptions(),
		onSuccess: async () => {
			toast.success("Overlay saved (replace)");
			if (overlayOrgId) {
				await qc.invalidateQueries({
					queryKey: [
						"regions",
						regionId,
						"organizations",
						overlayOrgId,
						"overlay",
					],
				});
			}
			setOverlayOrgId(null);
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const delOverlay = useMutation({
		...deleteOverlayMutationOptions(),
		onSuccess: async () => {
			toast.success("Overlay removed (inherit base)");
			setOverlayOrgId(null);
			await qc.invalidateQueries({
				queryKey: ["regions", regionId, "organizations"],
			});
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const orgName = new Map((orgsQ.data ?? []).map((o) => [o.id, o.name]));
	const boundIds = new Set((memQ.data ?? []).map((m) => m.organization_id));
	const unboundOrgs = (orgsQ.data ?? []).filter((o) => !boundIds.has(o.id));

	if (!isAdmin) {
		return (
			<>
				<PageHeader title="Region" />
				<PageBody>
					<p className="text-muted-foreground text-sm">
						Platform admin access is required.
					</p>
				</PageBody>
			</>
		);
	}

	return (
		<>
			<PageHeader
				title={regionQ.data?.name ?? "Region"}
				description={
					regionQ.data
						? `Slug ${regionQ.data.slug} · deployments, org bindings, overlays`
						: "Manage gateway deployments"
				}
				actions={
					<div className="flex items-center gap-2">
						{regionQ.data ? (
							<Select
								value={regionQ.data.status}
								onValueChange={(status) => update.mutate({ regionId, status })}
							>
								<SelectTrigger className="w-[140px]">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="active">active</SelectItem>
									<SelectItem value="draining">draining</SelectItem>
									<SelectItem value="disabled">disabled</SelectItem>
								</SelectContent>
							</Select>
						) : null}
						<Button onClick={() => setOpen(true)}>
							<PlusIcon />
							Register deployment
						</Button>
					</div>
				}
			/>
			<PageBody className="space-y-8">
				{joinToken ? (
					<div className="rounded-md border bg-muted/40 p-4 text-sm">
						<p className="font-medium">Join token (shown once)</p>
						<p className="mt-1 break-all font-mono text-xs">{joinToken}</p>
						<p className="text-muted-foreground mt-2">
							Set <code>AFI_DEPLOYMENT_JOIN_TOKEN</code> and{" "}
							<code>AFI_DEPLOYMENT_ID</code> on the gateway. Use region slug{" "}
							<code>{regionQ.data?.slug}</code> as <code>AFI_REGION_ID</code>{" "}
							for object-store snapshots.
						</p>
						<Button
							variant="ghost"
							size="sm"
							className="mt-2"
							onClick={() => setJoinToken(null)}
						>
							Dismiss
						</Button>
					</div>
				) : null}

				<section className="space-y-3">
					<div className="flex items-center justify-between gap-2">
						<div>
							<h2 className="font-medium text-sm">Organization bindings</h2>
							<p className="text-muted-foreground text-xs">
								Only bound orgs appear in this region&apos;s snapshot. Overlay
								replace overrides base gateway config for that org here.
							</p>
						</div>
						<Button
							variant="outline"
							size="sm"
							disabled={bindAll.isPending}
							onClick={() => bindAll.mutate({ regionId })}
						>
							Bind all orgs
						</Button>
						<Button
							variant="outline"
							size="sm"
							onClick={() => setBindOpen(true)}
						>
							<PlusIcon />
							Bind org
						</Button>
					</div>
					<QueryGate
						isPending={memQ.isPending}
						isError={memQ.isError}
						error={memQ.error}
						onRetry={() => void memQ.refetch()}
					>
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Organization</TableHead>
										<TableHead>Status</TableHead>
										<TableHead className="w-40" />
									</TableRow>
								</TableHeader>
								<TableBody>
									{(memQ.data ?? []).length === 0 ? (
										<TableRow>
											<TableCell colSpan={3} className="text-muted-foreground">
												No organizations bound. Unbound orgs are omitted from
												regional snapshots.
											</TableCell>
										</TableRow>
									) : (
										(memQ.data ?? []).map((m) => (
											<TableRow key={m.organization_id}>
												<TableCell>
													<div className="font-medium">
														{orgName.get(m.organization_id) ??
															m.organization_id}
													</div>
													<div className="text-muted-foreground font-mono text-xs">
														{m.organization_id}
													</div>
												</TableCell>
												<TableCell>
													<Badge variant={statusVariant(m.status)}>
														{m.status}
													</Badge>
												</TableCell>
												<TableCell className="text-right">
													<div className="flex justify-end gap-2">
														<Button
															variant="ghost"
															size="sm"
															onClick={() => setOverlayOrgId(m.organization_id)}
														>
															Overlay…
														</Button>
														<Button
															variant="outline"
															size="sm"
															onClick={() =>
																unbind.mutate({
																	regionId,
																	orgId: m.organization_id,
																})
															}
														>
															Unbind
														</Button>
													</div>
												</TableCell>
											</TableRow>
										))
									)}
								</TableBody>
							</Table>
						</div>
					</QueryGate>
				</section>

				<section className="space-y-3">
					<h2 className="font-medium text-sm">Deployments</h2>
					<QueryGate
						isPending={depsQ.isPending}
						isError={depsQ.isError}
						error={depsQ.error}
						onRetry={() => void depsQ.refetch()}
					>
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Name</TableHead>
										<TableHead>Status</TableHead>
										<TableHead>Snapshot</TableHead>
										<TableHead>Last seen</TableHead>
										<TableHead className="w-40" />
									</TableRow>
								</TableHeader>
								<TableBody>
									{(depsQ.data ?? []).length === 0 ? (
										<TableRow>
											<TableCell colSpan={5} className="text-muted-foreground">
												No deployments registered.
											</TableCell>
										</TableRow>
									) : (
										(depsQ.data ?? []).map((d) => (
											<TableRow key={d.id}>
												<TableCell>
													<div className="font-medium">{d.name}</div>
													<div className="text-muted-foreground font-mono text-xs">
														{d.id}
													</div>
												</TableCell>
												<TableCell>
													<Badge variant={statusVariant(d.status)}>
														{d.status}
													</Badge>
												</TableCell>
												<TableCell className="font-mono text-sm">
													{d.reported_snapshot_version || "—"}
												</TableCell>
												<TableCell className="text-sm">
													{d.last_seen_at
														? new Date(d.last_seen_at).toLocaleString()
														: "—"}
												</TableCell>
												<TableCell className="text-right">
													<div className="flex justify-end gap-2">
														<Button
															variant="outline"
															size="sm"
															onClick={() =>
																rotate.mutate({
																	regionId,
																	deploymentId: d.id,
																})
															}
														>
															Rotate token
														</Button>
													</div>
												</TableCell>
											</TableRow>
										))
									)}
								</TableBody>
							</Table>
						</div>
					</QueryGate>
				</section>
			</PageBody>

			<Sheet open={open} onOpenChange={setOpen}>
				<SheetContent className="overflow-y-auto sm:max-w-md">
					<SheetHeader>
						<SheetTitle>Register deployment</SheetTitle>
						<SheetDescription>
							Creates a join token for a regional gateway spoke.
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!name.trim() || register.isPending) return;
							register.mutate({
								regionId,
								name: name.trim(),
								public_base_url: baseURL.trim() || undefined,
							});
						}}
					>
						<div className="grid gap-2">
							<Label htmlFor="dep-name">Name</Label>
							<Input
								id="dep-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="gw-eu-1"
								autoFocus
							/>
						</div>
						<div className="grid gap-2">
							<Label htmlFor="dep-url">Public base URL (optional)</Label>
							<Input
								id="dep-url"
								value={baseURL}
								onChange={(e) => setBaseURL(e.target.value)}
								placeholder="https://gw.eu.example.com"
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
								disabled={!name.trim() || register.isPending}
							>
								{register.isPending ? "Registering…" : "Register"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>

			<Sheet open={bindOpen} onOpenChange={setBindOpen}>
				<SheetContent className="overflow-y-auto sm:max-w-md">
					<SheetHeader>
						<SheetTitle>Bind organization</SheetTitle>
						<SheetDescription>
							Adds the org to this region&apos;s snapshot allowlist.
						</SheetDescription>
					</SheetHeader>
					<form
						className="flex flex-1 flex-col gap-4 px-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!bindOrgId || bind.isPending) return;
							bind.mutate({ regionId, organization_id: bindOrgId });
						}}
					>
						<div className="grid gap-2">
							<Label>Organization</Label>
							<Select value={bindOrgId} onValueChange={setBindOrgId}>
								<SelectTrigger>
									<SelectValue placeholder="Select org" />
								</SelectTrigger>
								<SelectContent>
									{unboundOrgs.map((o) => (
										<SelectItem key={o.id} value={o.id}>
											{o.name}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
						<SheetFooter className="gap-2 px-0 sm:flex-col sm:space-x-0">
							<Button
								type="button"
								variant="outline"
								onClick={() => setBindOpen(false)}
							>
								Cancel
							</Button>
							<Button type="submit" disabled={!bindOrgId || bind.isPending}>
								{bind.isPending ? "Binding…" : "Bind"}
							</Button>
						</SheetFooter>
					</form>
				</SheetContent>
			</Sheet>

			{overlayOrgId ? (
				<RegionOverlaySheet
					regionId={regionId}
					orgId={overlayOrgId}
					orgLabel={orgName.get(overlayOrgId) ?? overlayOrgId}
					onClose={() => setOverlayOrgId(null)}
					onSave={(payload) =>
						putOverlay.mutate({ regionId, orgId: overlayOrgId, payload })
					}
					onInherit={() => delOverlay.mutate({ regionId, orgId: overlayOrgId })}
					saving={putOverlay.isPending}
					inheriting={delOverlay.isPending}
				/>
			) : null}
		</>
	);
}
