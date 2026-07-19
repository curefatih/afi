import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { PlusIcon, ShieldIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import {
	assignCredentialMutationOptions,
	type Credential,
	type CredentialAssignment,
	type CredentialStorageKind,
	createCredentialMutationOptions,
	credentialAssignmentsQueryOptions,
	credentialsQueryOptions,
	deleteCredentialAssignmentMutationOptions,
	deleteCredentialMutationOptions,
} from "#/api/credentials";
import { orgMembersQueryOptions } from "#/api/organization";
import { PROVIDER_TYPE_PRESETS } from "#/api/provider";
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
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/secrets")({
	...pageTitle("Secrets"),
	component: RouteComponent,
});

const PROVIDER_TYPES = Object.keys(PROVIDER_TYPE_PRESETS).filter(
	(t) => t !== "echo",
);

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const credentials = useQuery(credentialsQueryOptions(orgId));
	const assignments = useQuery(credentialAssignmentsQueryOptions(orgId));
	const members = useQuery(orgMembersQueryOptions(orgId));

	const [createOpen, setCreateOpen] = useState(false);
	const [assignFor, setAssignFor] = useState<Credential | null>(null);

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const projectName = useMemo(() => {
		const map = new Map((org?.projects ?? []).map((p) => [p.id, p.name]));
		return (id: string) => map.get(id) ?? id;
	}, [org?.projects]);

	const assignmentsByCred = useMemo(() => {
		const map = new Map<string, CredentialAssignment[]>();
		for (const a of assignments.data ?? []) {
			const list = map.get(a.credential_id) ?? [];
			list.push(a);
			map.set(a.credential_id, list);
		}
		return map;
	}, [assignments.data]);

	const invalidate = () => {
		void qc.invalidateQueries({
			queryKey: ["organizations", orgId, "credentials"],
		});
		void qc.invalidateQueries({
			queryKey: ["organizations", orgId, "credential-assignments"],
		});
	};

	const create = useMutation({
		...createCredentialMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("Credential created");
			setCreateOpen(false);
		},
		onError: (e: Error) => toast.error(e.message),
	});
	const del = useMutation({
		...deleteCredentialMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("Credential deleted");
		},
		onError: (e: Error) => toast.error(e.message),
	});
	const assign = useMutation({
		...assignCredentialMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("Credential assigned");
			setAssignFor(null);
		},
		onError: (e: Error) => toast.error(e.message),
	});
	const unassign = useMutation({
		...deleteCredentialAssignmentMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("Assignment removed");
		},
		onError: (e: Error) => toast.error(e.message),
	});

	return (
		<>
			<PageHeader
				title="Secrets"
				description="Provider credentials (env refs or encrypted) assigned to the org or a project. Virtual keys inherit the most specific assignment; providers.api_key_env remains a migration fallback."
				actions={
					isOrgAdmin ? (
						<Button onClick={() => setCreateOpen(true)}>
							<PlusIcon data-icon="inline-start" />
							Add credential
						</Button>
					) : null
				}
			/>
			<PageBody>
				<QueryGate query={credentials}>
					{(list) =>
						list.length === 0 ? (
							<Empty>
								<EmptyHeader>
									<EmptyMedia variant="icon">
										<ShieldIcon />
									</EmptyMedia>
									<EmptyTitle>No credentials yet</EmptyTitle>
									<EmptyDescription>
										Create an OpenAI (or other) credential and assign it to this
										organization or a project.
									</EmptyDescription>
								</EmptyHeader>
								{isOrgAdmin ? (
									<EmptyContent>
										<Button onClick={() => setCreateOpen(true)}>
											Add credential
										</Button>
									</EmptyContent>
								) : null}
							</Empty>
						) : (
							<div className="rounded-lg border">
								<Table>
									<TableHeader>
										<TableRow>
											<TableHead>Name</TableHead>
											<TableHead>Provider</TableHead>
											<TableHead>Storage</TableHead>
											<TableHead>Assignments</TableHead>
											<TableHead>Status</TableHead>
											{isOrgAdmin ? <TableHead className="w-[1%]" /> : null}
										</TableRow>
									</TableHeader>
									<TableBody>
										{list.map((c) => {
											const asgs = assignmentsByCred.get(c.id) ?? [];
											return (
												<TableRow key={c.id}>
													<TableCell className="font-medium">
														{c.name}
														{c.storage_kind === "env" && c.secret_ref ? (
															<div className="text-muted-foreground text-xs font-normal">
																{c.secret_ref}
															</div>
														) : null}
													</TableCell>
													<TableCell>{c.provider_type}</TableCell>
													<TableCell>
														<Badge variant="outline" className="font-normal">
															{c.storage_kind === "encrypted_db"
																? "encrypted"
																: "env"}
														</Badge>
													</TableCell>
													<TableCell>
														<div className="flex flex-col gap-1">
															{asgs.length === 0 ? (
																<span className="text-muted-foreground text-sm">
																	—
																</span>
															) : (
																asgs.map((a) => (
																	<div
																		key={a.id}
																		className="flex items-center gap-2 text-sm"
																	>
																		<span>
																			{a.scope_type === "organization"
																				? `Org · ${org?.name ?? a.scope_id}`
																				: `Project · ${projectName(a.scope_id)}`}
																		</span>
																		{isOrgAdmin ? (
																			<Button
																				variant="ghost"
																				size="sm"
																				className="h-6 px-2 text-xs"
																				disabled={unassign.isPending}
																				onClick={() => unassign.mutate(a.id)}
																			>
																				Remove
																			</Button>
																		) : null}
																	</div>
																))
															)}
														</div>
													</TableCell>
													<TableCell>
														<Badge
															variant={
																c.status === "active" ? "secondary" : "outline"
															}
															className="font-normal"
														>
															{c.status}
														</Badge>
													</TableCell>
													{isOrgAdmin ? (
														<TableCell className="text-right whitespace-nowrap">
															<Button
																variant="outline"
																size="sm"
																onClick={() => setAssignFor(c)}
															>
																Assign
															</Button>{" "}
															<Button
																variant="ghost"
																size="sm"
																disabled={del.isPending}
																onClick={() => {
																	if (
																		confirm(
																			`Delete credential “${c.name}”? Remove assignments first if delete fails.`,
																		)
																	) {
																		del.mutate(c.id);
																	}
																}}
															>
																Delete
															</Button>
														</TableCell>
													) : null}
												</TableRow>
											);
										})}
									</TableBody>
								</Table>
							</div>
						)
					}
				</QueryGate>
			</PageBody>

			<CreateCredentialSheet
				open={createOpen}
				onOpenChange={setCreateOpen}
				pending={create.isPending}
				onSubmit={(input) => create.mutate({ orgId, ...input })}
			/>
			<AssignCredentialSheet
				credential={assignFor}
				orgId={orgId}
				orgName={org?.name ?? "Organization"}
				projects={org?.projects ?? []}
				pending={assign.isPending}
				onOpenChange={(open) => {
					if (!open) setAssignFor(null);
				}}
				onSubmit={(scope_type, scope_id) => {
					if (!assignFor) return;
					assign.mutate({
						orgId,
						credential_id: assignFor.id,
						scope_type,
						scope_id,
					});
				}}
			/>
		</>
	);
}

function CreateCredentialSheet({
	open,
	onOpenChange,
	pending,
	onSubmit,
}: {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	pending: boolean;
	onSubmit: (input: {
		name: string;
		provider_type: string;
		storage_kind: CredentialStorageKind;
		secret_ref?: string;
		secret_value?: string;
	}) => void;
}) {
	const [name, setName] = useState("");
	const [providerType, setProviderType] = useState("openai");
	const [storageKind, setStorageKind] = useState<CredentialStorageKind>("env");
	const [secretRef, setSecretRef] = useState(
		PROVIDER_TYPE_PRESETS.openai.api_key_env,
	);
	const [secretValue, setSecretValue] = useState("");

	const reset = () => {
		setName("");
		setProviderType("openai");
		setStorageKind("env");
		setSecretRef(PROVIDER_TYPE_PRESETS.openai.api_key_env);
		setSecretValue("");
	};

	return (
		<Sheet
			open={open}
			onOpenChange={(v) => {
				if (!v) reset();
				onOpenChange(v);
			}}
		>
			<SheetContent className="flex flex-col sm:max-w-md">
				<SheetHeader>
					<SheetTitle>Add credential</SheetTitle>
					<SheetDescription>
						Env storage references a gateway process variable. Encrypted storage
						seals the secret with AFI_CREDENTIALS_MASTER_KEY.
					</SheetDescription>
				</SheetHeader>
				<div className="flex flex-1 flex-col gap-4 px-4">
					<div className="grid gap-2">
						<Label htmlFor="cred-name">Name</Label>
						<Input
							id="cred-name"
							value={name}
							onChange={(e) => setName(e.target.value)}
							placeholder="OpenAI — primary"
						/>
					</div>
					<div className="grid gap-2">
						<Label>Provider type</Label>
						<Select
							value={providerType}
							onValueChange={(v) => {
								const next = v ?? "openai";
								setProviderType(next);
								const preset = PROVIDER_TYPE_PRESETS[next];
								if (preset && storageKind === "env") {
									setSecretRef(preset.api_key_env);
								}
							}}
						>
							<SelectTrigger className="w-full">
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								{PROVIDER_TYPES.map((t) => (
									<SelectItem key={t} value={t}>
										{PROVIDER_TYPE_PRESETS[t]?.name ?? t}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>
					<div className="grid gap-2">
						<Label>Storage</Label>
						<Select
							value={storageKind}
							onValueChange={(v) =>
								setStorageKind((v as CredentialStorageKind) ?? "env")
							}
						>
							<SelectTrigger className="w-full">
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="env">Environment variable</SelectItem>
								<SelectItem value="encrypted_db">
									Stored secret (encrypted)
								</SelectItem>
							</SelectContent>
						</Select>
					</div>
					{storageKind === "env" ? (
						<div className="grid gap-2">
							<Label htmlFor="cred-ref">Env var name</Label>
							<Input
								id="cred-ref"
								value={secretRef}
								onChange={(e) => setSecretRef(e.target.value)}
								placeholder="OPENAI_API_KEY"
							/>
						</div>
					) : (
						<div className="grid gap-2">
							<Label htmlFor="cred-value">API key</Label>
							<Input
								id="cred-value"
								type="password"
								value={secretValue}
								onChange={(e) => setSecretValue(e.target.value)}
								placeholder="sk-…"
								autoComplete="off"
							/>
						</div>
					)}
				</div>
				<SheetFooter>
					<Button variant="outline" onClick={() => onOpenChange(false)}>
						Cancel
					</Button>
					<Button
						disabled={
							pending ||
							!name.trim() ||
							(storageKind === "env" ? !secretRef.trim() : !secretValue.trim())
						}
						onClick={() =>
							onSubmit({
								name: name.trim(),
								provider_type: providerType,
								storage_kind: storageKind,
								secret_ref:
									storageKind === "env" ? secretRef.trim() : undefined,
								secret_value:
									storageKind === "encrypted_db"
										? secretValue.trim()
										: undefined,
							})
						}
					>
						Create
					</Button>
				</SheetFooter>
			</SheetContent>
		</Sheet>
	);
}

function AssignCredentialSheet({
	credential,
	orgId,
	orgName,
	projects,
	pending,
	onOpenChange,
	onSubmit,
}: {
	credential: Credential | null;
	orgId: string;
	orgName: string;
	projects: { id: string; name: string }[];
	pending: boolean;
	onOpenChange: (open: boolean) => void;
	onSubmit: (scopeType: "organization" | "project", scopeId: string) => void;
}) {
	const [scopeType, setScopeType] = useState<"organization" | "project">(
		"organization",
	);
	const [projectId, setProjectId] = useState("");

	return (
		<Sheet open={!!credential} onOpenChange={onOpenChange}>
			<SheetContent className="flex flex-col sm:max-w-md">
				<SheetHeader>
					<SheetTitle>Assign credential</SheetTitle>
					<SheetDescription>
						{credential
							? `Bind “${credential.name}” (${credential.provider_type}) to a scope. Project overrides organization.`
							: null}
					</SheetDescription>
				</SheetHeader>
				<div className="flex flex-1 flex-col gap-4 px-4">
					<div className="grid gap-2">
						<Label>Scope</Label>
						<Select
							value={scopeType}
							onValueChange={(v) =>
								setScopeType(
									(v as "organization" | "project") ?? "organization",
								)
							}
						>
							<SelectTrigger className="w-full">
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="organization">Organization</SelectItem>
								<SelectItem value="project">Project</SelectItem>
							</SelectContent>
						</Select>
					</div>
					{scopeType === "organization" ? (
						<p className="text-muted-foreground text-sm">
							Assigned to <span className="text-foreground">{orgName}</span>
						</p>
					) : (
						<div className="grid gap-2">
							<Label>Project</Label>
							<Select
								value={projectId || null}
								onValueChange={(v) => setProjectId(v ?? "")}
							>
								<SelectTrigger className="w-full">
									<SelectValue placeholder="Select project" />
								</SelectTrigger>
								<SelectContent>
									{projects.map((p) => (
										<SelectItem key={p.id} value={p.id}>
											{p.name}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
					)}
				</div>
				<SheetFooter>
					<Button variant="outline" onClick={() => onOpenChange(false)}>
						Cancel
					</Button>
					<Button
						disabled={
							pending || (scopeType === "project" && !projectId) || !credential
						}
						onClick={() =>
							onSubmit(
								scopeType,
								scopeType === "organization" ? orgId : projectId,
							)
						}
					>
						Assign
					</Button>
				</SheetFooter>
			</SheetContent>
		</Sheet>
	);
}
