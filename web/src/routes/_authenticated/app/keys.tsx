import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { KeyRoundIcon, PlusIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import {
	type ApiKey,
	type KeyKind,
	deleteKeyMutationOptions,
	orgKeysQueryOptions,
} from "#/api/keys";
import { orgMembersQueryOptions } from "#/api/organization";
import { CreateKeySheet } from "#/components/create-key-sheet";
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
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "#/components/ui/tabs";
import { useOrgBootstrap } from "#/hooks/use-org-bootstrap";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/keys")({
	staticData: {
		getTitle: () => "API Keys",
	},
	component: RouteComponent,
});

type KeyTab = "all" | KeyKind;

function formatKeyPrefix(prefix: string) {
	if (!prefix) return "••••••••";
	return `${prefix}…`;
}

function normalizeKind(kind?: string): KeyKind {
	return kind === "personal" ? "personal" : "service_account";
}

function kindLabel(kind: KeyKind) {
	return kind === "personal" ? "Personal" : "Service account";
}

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const user = useAuthUser();
	const { isBootstrapping, isError, error, refetch } = useOrgBootstrap();
	const [open, setOpen] = useState(false);
	const [tab, setTab] = useState<KeyTab>("all");
	const [tabReady, setTabReady] = useState(false);
	const qc = useQueryClient();

	const members = useQuery(orgMembersQueryOptions(orgId));
	const keys = useQuery(orgKeysQueryOptions(orgId));

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const ownerEmail = useMemo(() => {
		const map = new Map(
			(members.data ?? []).map((m) => [m.user_id, m.email] as const),
		);
		return (id?: string) => (id ? (map.get(id) ?? id) : "—");
	}, [members.data]);

	const projectName = useMemo(() => {
		const map = new Map(
			(activeOrg?.projects ?? []).map((p) => [p.id, p.name] as const),
		);
		return (id?: string) => {
			if (!id) return "Org-wide";
			return map.get(id) ?? id;
		};
	}, [activeOrg?.projects]);

	const allKeys = useMemo(
		() =>
			(keys.data ?? []).map((k) => ({
				...k,
				kind: normalizeKind(k.kind),
			})),
		[keys.data],
	);

	const personal = allKeys.filter((k) => k.kind === "personal");
	const service = allKeys.filter((k) => k.kind === "service_account");

	// Prefer showing existing keys on first load (seed key is service_account).
	useEffect(() => {
		if (tabReady || keys.isLoading || !keys.isSuccess) return;
		if (personal.length === 0 && service.length > 0) {
			setTab("service_account");
		} else {
			setTab("all");
		}
		setTabReady(true);
	}, [tabReady, keys.isLoading, keys.isSuccess, personal.length, service.length]);

	const del = useMutation({
		...deleteKeyMutationOptions(),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["organizations", orgId, "keys"] });
			toast.success("Key revoked");
		},
		onError: (err) => toast.error(err.message || "Failed to revoke key"),
	});

	const canRevoke = (k: ApiKey) =>
		isOrgAdmin ||
		(normalizeKind(k.kind) === "personal" && k.owner_user_id === user?.id);

	const rowsForTab = (t: KeyTab) => {
		if (t === "all") return allKeys;
		if (t === "personal") return personal;
		return service;
	};

	const renderTable = (
		rows: ApiKey[],
		emptyTitle: string,
		emptyDesc: string,
		createKind: KeyKind,
	) =>
		rows.length === 0 ? (
			<Empty className="border min-h-48">
				<EmptyHeader>
					<EmptyMedia variant="icon">
						<KeyRoundIcon />
					</EmptyMedia>
					<EmptyTitle>{emptyTitle}</EmptyTitle>
					<EmptyDescription>{emptyDesc}</EmptyDescription>
				</EmptyHeader>
				<EmptyContent>
					<Button
						onClick={() => {
							setTab(createKind);
							setOpen(true);
						}}
						disabled={createKind === "service_account" && !isOrgAdmin}
					>
						<PlusIcon />
						Create key
					</Button>
				</EmptyContent>
			</Empty>
		) : (
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead>Name</TableHead>
						<TableHead>Kind</TableHead>
						<TableHead>Owner / scope</TableHead>
						<TableHead>Key</TableHead>
						<TableHead>Created</TableHead>
						<TableHead className="w-24" />
					</TableRow>
				</TableHeader>
				<TableBody>
					{rows.map((row) => {
						const kind = normalizeKind(row.kind);
						return (
							<TableRow key={row.id}>
								<TableCell className="font-medium">{row.name}</TableCell>
								<TableCell>
									<Badge variant="secondary">{kindLabel(kind)}</Badge>
								</TableCell>
								<TableCell className="text-muted-foreground text-sm">
									{kind === "personal"
										? ownerEmail(row.owner_user_id)
										: projectName(row.project_id)}
								</TableCell>
								<TableCell>
									<Badge variant="outline" className="font-mono">
										{formatKeyPrefix(row.key_prefix)}
									</Badge>
								</TableCell>
								<TableCell className="text-muted-foreground">
									{new Date(row.created_at).toLocaleString()}
								</TableCell>
								<TableCell>
									{canRevoke(row) ? (
										<Button
											variant="outline"
											size="sm"
											disabled={del.isPending}
											onClick={() => del.mutate(row.id)}
										>
											Revoke
										</Button>
									) : null}
								</TableCell>
							</TableRow>
						);
					})}
				</TableBody>
			</Table>
		);

	const keysLoading = !!orgId && keys.isLoading;
	const membersLoading = !!orgId && members.isLoading;

	return (
		<PageBody>
			<PageHeader
				title="API Keys"
				description="Personal keys authenticate as you. Service accounts are for automation — org- or project-scoped."
				actions={
					<Button onClick={() => setOpen(true)} disabled={!orgId}>
						<PlusIcon />
						New key
					</Button>
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
				<Tabs
					value={tab}
					onValueChange={(v) => setTab((v as KeyTab) ?? "all")}
				>
					<TabsList>
						<TabsTrigger value="all">All ({allKeys.length})</TabsTrigger>
						<TabsTrigger value="personal">
							Personal ({personal.length})
						</TabsTrigger>
						<TabsTrigger value="service_account">
							Service accounts ({service.length})
						</TabsTrigger>
					</TabsList>
					<TabsContent value="all" className="mt-4">
						{renderTable(
							rowsForTab("all"),
							"No API keys",
							"Create a personal or service-account key to authenticate gateway traffic.",
							"personal",
						)}
					</TabsContent>
					<TabsContent value="personal" className="mt-4">
						{renderTable(
							personal,
							"No personal keys",
							"Create a personal key to call the gateway as yourself.",
							"personal",
						)}
					</TabsContent>
					<TabsContent value="service_account" className="mt-4">
						{renderTable(
							service,
							"No service account keys",
							isOrgAdmin
								? "Create an org-wide or project service account for automation."
								: "Ask an org admin to create a service account key.",
							"service_account",
						)}
					</TabsContent>
				</Tabs>
			</QueryGate>

			<CreateKeySheet
				open={open}
				onOpenChange={setOpen}
				defaultKind={tab === "service_account" ? "service_account" : "personal"}
				isOrgAdmin={isOrgAdmin}
			/>
		</PageBody>
	);
}
