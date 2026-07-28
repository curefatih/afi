import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { FingerprintIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { orgMembersQueryOptions } from "#/api/organization";
import {
	deleteSigningKeyMutationOptions,
	orgSigningKeysQueryOptions,
	type SigningKey,
	updateSigningKeyMutationOptions,
} from "#/api/signing-keys";
import { CreateSigningKeySheet } from "#/components/create-signing-key-sheet";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { RotateSigningKeySheet } from "#/components/rotate-signing-key-sheet";
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
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { useOrgBootstrap } from "#/hooks/use-org-bootstrap";
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/signing-keys")({
	...pageTitle("Signing Keys"),
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const user = useAuthUser();
	const { isBootstrapping, isError, error, refetch } = useOrgBootstrap();
	const [createOpen, setCreateOpen] = useState(false);
	const [rotateKey, setRotateKey] = useState<SigningKey | null>(null);
	const qc = useQueryClient();

	const members = useQuery(orgMembersQueryOptions(orgId));
	const keys = useQuery(orgSigningKeysQueryOptions(orgId));

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const projectName = useMemo(() => {
		const map = new Map(
			(activeOrg?.projects ?? []).map((p) => [p.id, p.name] as const),
		);
		return (id?: string) => {
			if (!id) return "Org-wide";
			return map.get(id) ?? id;
		};
	}, [activeOrg?.projects]);

	const invalidate = () => {
		void qc.invalidateQueries({
			queryKey: ["organizations", orgId, "signing-keys"],
		});
	};

	const update = useMutation({
		...updateSigningKeyMutationOptions(),
		onSuccess: (_data, vars) => {
			invalidate();
			toast.success(
				vars.status === "disabled"
					? "Signing key disabled"
					: "Signing key enabled",
			);
		},
		onError: (err: Error) =>
			toast.error(err.message || "Failed to update signing key"),
	});

	const del = useMutation({
		...deleteSigningKeyMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("Signing key deleted");
		},
		onError: (err: Error) =>
			toast.error(err.message || "Failed to delete signing key"),
	});

	const rows = keys.data ?? [];
	const keysLoading = !!orgId && keys.isLoading;
	const membersLoading = !!orgId && members.isLoading;
	const pending = update.isPending || del.isPending;

	return (
		<PageBody>
			<PageHeader
				title="Signing Keys"
				description="Register Ed25519 public keys for signed gateway requests."
				info="Services sign each request with their private key and send X-AFI-* headers. This is an alternative to virtual API keys."
				actions={
					isOrgAdmin ? (
						<Button onClick={() => setCreateOpen(true)} disabled={!orgId}>
							<PlusIcon />
							Register key
						</Button>
					) : null
				}
			/>

			<QueryGate
				isPending={isBootstrapping || keysLoading || membersLoading}
				isError={isError || keys.isError}
				error={error || keys.error}
				onRetry={() => {
					refetch();
					void keys.refetch();
				}}
			>
				{rows.length === 0 ? (
					<Empty className="border min-h-48">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<FingerprintIcon />
							</EmptyMedia>
							<EmptyTitle>No signing keys</EmptyTitle>
							<EmptyDescription>
								{isOrgAdmin
									? "Register a public key so services can authenticate with signed requests."
									: "Ask an org admin to register a signing key for service authentication."}
							</EmptyDescription>
						</EmptyHeader>
						{isOrgAdmin ? (
							<EmptyContent>
								<Button onClick={() => setCreateOpen(true)}>
									<PlusIcon />
									Register key
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Key ID</TableHead>
								<TableHead>Scope</TableHead>
								<TableHead>Algorithm</TableHead>
								<TableHead>Status</TableHead>
								<TableHead>Updated</TableHead>
								<TableHead className="w-56" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{rows.map((row) => (
								<TableRow key={row.id}>
									<TableCell className="font-medium">{row.name}</TableCell>
									<TableCell>
										<Badge variant="outline" className="font-mono">
											{row.key_id}
										</Badge>
									</TableCell>
									<TableCell className="text-muted-foreground text-sm">
										{projectName(row.project_id)}
									</TableCell>
									<TableCell className="font-mono text-sm uppercase">
										{row.algorithm}
									</TableCell>
									<TableCell>
										<Badge
											variant={
												row.status === "active" ? "secondary" : "outline"
											}
										>
											{row.status}
										</Badge>
									</TableCell>
									<TableCell className="text-muted-foreground">
										{new Date(row.updated_at).toLocaleString()}
									</TableCell>
									<TableCell>
										{isOrgAdmin ? (
											<div className="flex flex-wrap justify-end gap-2">
												<Button
													variant="outline"
													size="sm"
													disabled={pending}
													onClick={() => setRotateKey(row)}
												>
													Rotate
												</Button>
												<Button
													variant="outline"
													size="sm"
													disabled={pending}
													onClick={() =>
														update.mutate({
															signingKeyId: row.id,
															status:
																row.status === "active" ? "disabled" : "active",
														})
													}
												>
													{row.status === "active" ? "Disable" : "Enable"}
												</Button>
												<Button
													variant="outline"
													size="sm"
													disabled={pending}
													onClick={() => del.mutate(row.id)}
												>
													Delete
												</Button>
											</div>
										) : null}
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				)}
			</QueryGate>

			<CreateSigningKeySheet open={createOpen} onOpenChange={setCreateOpen} />
			<RotateSigningKeySheet
				open={!!rotateKey}
				onOpenChange={(next) => {
					if (!next) setRotateKey(null);
				}}
				signingKey={rotateKey}
			/>
		</PageBody>
	);
}
