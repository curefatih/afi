import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { NetworkIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import {
	type ControlPlanePeer,
	type FederationPeerWithToken,
	federationPeersQueryOptions,
	registerFederationPeerMutationOptions,
	rotateFederationPeerTokenMutationOptions,
	updateFederationPeerMutationOptions,
} from "#/api/federation";
import { regionsQueryOptions } from "#/api/regions";
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

export const Route = createFileRoute("/_authenticated/app/federation/")({
	...pageTitle("Federation"),
	component: RouteComponent,
});

function RouteComponent() {
	const user = useAuthUser();
	const isAdmin = user?.role === "admin";
	const qc = useQueryClient();
	const q = useQuery(federationPeersQueryOptions());
	const regionsQ = useQuery(regionsQueryOptions());
	const [open, setOpen] = useState(false);
	const [name, setName] = useState("");
	const [regionId, setRegionId] = useState("");
	const [baseUrl, setBaseUrl] = useState("");
	const [tokenOnce, setTokenOnce] = useState<FederationPeerWithToken | null>(
		null,
	);

	const register = useMutation({
		...registerFederationPeerMutationOptions(),
		onSuccess: async (created) => {
			toast.success("Peer registered — copy the join token now");
			setTokenOnce(created);
			setOpen(false);
			setName("");
			setRegionId("");
			setBaseUrl("");
			await qc.invalidateQueries({ queryKey: ["federation", "peers"] });
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const disable = useMutation({
		...updateFederationPeerMutationOptions(),
		onSuccess: async () => {
			toast.success("Peer updated");
			await qc.invalidateQueries({ queryKey: ["federation", "peers"] });
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const rotate = useMutation({
		...rotateFederationPeerTokenMutationOptions(),
		onSuccess: async (out) => {
			toast.success("Token rotated — copy the new join token now");
			setTokenOnce(out);
			await qc.invalidateQueries({ queryKey: ["federation", "peers"] });
		},
		onError: (e: Error) => toast.error(e.message),
	});

	const peers = q.data ?? [];
	const regions = regionsQ.data ?? [];
	const regionName = (id: string) =>
		regions.find((r) => r.id === id)?.slug ?? id;

	if (!isAdmin) {
		return (
			<>
				<PageHeader title="Federation" />
				<PageBody>
					<p className="text-muted-foreground text-sm">
						Platform admin access required.
					</p>
				</PageBody>
			</>
		);
	}

	return (
		<>
			<PageHeader
				title="Federation"
				actions={
					<Button onClick={() => setOpen(true)}>
						<PlusIcon />
						Register peer
					</Button>
				}
			/>
			<PageBody>
				{tokenOnce ? (
					<div className="mb-6 rounded-md border border-amber-500/40 bg-amber-500/10 p-4 text-sm">
						<p className="font-medium">Join token (shown once)</p>
						<p className="mt-1 text-muted-foreground">
							Peer {tokenOnce.peer.name} ({tokenOnce.peer.id})
						</p>
						<code className="mt-2 block break-all rounded bg-background p-2 font-mono text-xs">
							{tokenOnce.join_token}
						</code>
						<Button
							className="mt-3"
							size="sm"
							variant="outline"
							onClick={() => setTokenOnce(null)}
						>
							Dismiss
						</Button>
					</div>
				) : null}

				<QueryGate query={q}>
					{peers.length === 0 ? (
						<Empty>
							<EmptyHeader>
								<EmptyMedia variant="icon">
									<NetworkIcon />
								</EmptyMedia>
								<EmptyTitle>No federation peers</EmptyTitle>
								<EmptyDescription>
									Register a regional control plane to pull memberships,
									overlays, and regional snapshots from this hub.
								</EmptyDescription>
							</EmptyHeader>
							<EmptyContent>
								<Button onClick={() => setOpen(true)}>
									<PlusIcon />
									Register peer
								</Button>
							</EmptyContent>
						</Empty>
					) : (
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Name</TableHead>
									<TableHead>Region</TableHead>
									<TableHead>Status</TableHead>
									<TableHead>Last sync</TableHead>
									<TableHead className="text-right">Actions</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								{peers.map((p: ControlPlanePeer) => (
									<TableRow key={p.id}>
										<TableCell className="font-medium">{p.name}</TableCell>
										<TableCell>{regionName(p.region_id)}</TableCell>
										<TableCell>
											<Badge variant="secondary">{p.status}</Badge>
										</TableCell>
										<TableCell className="text-muted-foreground text-xs">
											{p.last_sync_at
												? new Date(p.last_sync_at).toLocaleString()
												: "—"}
											{p.last_sync_error ? (
												<span className="mt-1 block text-destructive">
													{p.last_sync_error}
												</span>
											) : null}
										</TableCell>
										<TableCell className="space-x-2 text-right">
											<Button
												size="sm"
												variant="outline"
												onClick={() => rotate.mutate({ peerId: p.id })}
											>
												Rotate token
											</Button>
											{p.status !== "disabled" ? (
												<Button
													size="sm"
													variant="ghost"
													onClick={() =>
														disable.mutate({
															peerId: p.id,
															status: "disabled",
														})
													}
												>
													Disable
												</Button>
											) : (
												<Button
													size="sm"
													variant="ghost"
													onClick={() =>
														disable.mutate({
															peerId: p.id,
															status: "pending",
														})
													}
												>
													Re-enable
												</Button>
											)}
										</TableCell>
									</TableRow>
								))}
							</TableBody>
						</Table>
					)}
				</QueryGate>
			</PageBody>

			<Sheet open={open} onOpenChange={setOpen}>
				<SheetContent>
					<SheetHeader>
						<SheetTitle>Register federation peer</SheetTitle>
						<SheetDescription>
							Mints a join token for a regional control plane scoped to one
							region.
						</SheetDescription>
					</SheetHeader>
					<div className="flex flex-1 flex-col gap-4 px-4">
						<div className="space-y-2">
							<Label htmlFor="peer-name">Name</Label>
							<Input
								id="peer-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="EU regional CP"
							/>
						</div>
						<div className="space-y-2">
							<Label>Region</Label>
							<Select value={regionId} onValueChange={setRegionId}>
								<SelectTrigger>
									<SelectValue placeholder="Select region" />
								</SelectTrigger>
								<SelectContent>
									{regions.map((r) => (
										<SelectItem key={r.id} value={r.id}>
											{r.slug} — {r.name}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-2">
							<Label htmlFor="peer-url">Base URL (optional)</Label>
							<Input
								id="peer-url"
								value={baseUrl}
								onChange={(e) => setBaseUrl(e.target.value)}
								placeholder="https://cp.eu.example.com"
							/>
						</div>
					</div>
					<SheetFooter>
						<Button
							disabled={!name.trim() || !regionId || register.isPending}
							onClick={() =>
								register.mutate({
									name: name.trim(),
									region_id: regionId,
									base_url: baseUrl.trim() || undefined,
								})
							}
						>
							Register
						</Button>
					</SheetFooter>
				</SheetContent>
			</Sheet>
		</>
	);
}
