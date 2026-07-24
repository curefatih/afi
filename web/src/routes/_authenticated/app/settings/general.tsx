import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Building2Icon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import {
	type ObjectStoreConfig,
	orgDefaultRetryQueryOptions,
	orgMailQueryOptions,
	orgMembersQueryOptions,
	orgObjectStoreQueryOptions,
	testOrgMailMutationOptions,
	updateOrgDefaultRetryMutationOptions,
	updateOrgMailMutationOptions,
	updateOrgObjectStoreMutationOptions,
} from "#/api/organization";
import type { RetryConfig } from "#/api/routing";
import { CopyableId } from "#/components/copyable-id";
import { PageBody, PageHeader } from "#/components/page-header";
import {
	RetryEditor,
	toRetryPayload,
	validateRetry,
} from "#/components/routing/retry-editor";
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
import { Switch } from "#/components/ui/switch";
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/settings/general")({
	...pageTitle("Organization settings"),
	component: RouteComponent,
});

const emptyObjectStore = (): ObjectStoreConfig => ({
	enabled: false,
	endpoint: "",
	region: "us-east-1",
	bucket: "",
	use_ssl: false,
	path_style: true,
	credential_id: "",
	access_key_env: "",
	secret_key_env: "",
	presign_ttl_seconds: 3600,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const members = useQuery(orgMembersQueryOptions(orgId));
	const mail = useQuery({
		...orgMailQueryOptions(orgId),
		enabled: !!orgId,
	});
	const defaultRetry = useQuery({
		...orgDefaultRetryQueryOptions(orgId),
		enabled: !!orgId,
	});
	const objectStore = useQuery({
		...orgObjectStoreQueryOptions(orgId),
		enabled: !!orgId,
	});
	const updateMail = useMutation(updateOrgMailMutationOptions());
	const testMail = useMutation(testOrgMailMutationOptions());
	const updateRetry = useMutation(updateOrgDefaultRetryMutationOptions());
	const updateObjectStore = useMutation(updateOrgObjectStoreMutationOptions());
	const [retryDraft, setRetryDraft] = useState<RetryConfig | null>(null);
	const [retryLoaded, setRetryLoaded] = useState(false);
	const [storeDraft, setStoreDraft] = useState<ObjectStoreConfig>(
		emptyObjectStore(),
	);
	const [storeLoaded, setStoreLoaded] = useState(false);

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	useEffect(() => {
		if (!defaultRetry.isSuccess) return;
		setRetryDraft(defaultRetry.data.retry ?? null);
		setRetryLoaded(true);
	}, [defaultRetry.isSuccess, defaultRetry.data]);

	useEffect(() => {
		if (!objectStore.isSuccess) return;
		setStoreDraft(objectStore.data.object_store ?? emptyObjectStore());
		setStoreLoaded(true);
	}, [objectStore.isSuccess, objectStore.data]);

	const selected = mail.data?.selected || mail.data?.default_provider || "smtp";
	const enabled = mail.data?.enabled_providers ?? [];

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
				description={`Preferences for ${activeOrg.name}.`}
				info="Switch organizations from the sidebar or Organizations page."
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
						<dd>
							<CopyableId value={activeOrg.id} />
						</dd>
					</div>
				</dl>
			</section>

			{isOrgAdmin ? (
				<>
					<div className="grid gap-4 lg:grid-cols-2">
						<section className="space-y-3 rounded-md border p-4">
							<h2 className="text-sm font-medium">Default retry</h2>
							<p className="text-muted-foreground text-sm">
								Used for routes that do not set their own retry. Route-level
								retry always wins when configured.
							</p>
							{retryLoaded ? (
								<>
									<RetryEditor
										value={retryDraft}
										onChange={setRetryDraft}
										idPrefix="org-default-retry"
									/>
									<div className="flex justify-end">
										<Button
											disabled={updateRetry.isPending}
											onClick={() => {
												const err = validateRetry(retryDraft);
												if (err) {
													toast.error(err);
													return;
												}
												updateRetry.mutate(
													{ orgId, retry: toRetryPayload(retryDraft) },
													{
														onSuccess: (res) => {
															setRetryDraft(res.retry ?? null);
															void qc.invalidateQueries({
																queryKey: [
																	"organizations",
																	orgId,
																	"default-retry",
																],
															});
															toast.success("Default retry saved & published");
														},
														onError: (e) =>
															toast.error(
																e instanceof Error
																	? e.message
																	: "Failed to save default retry",
															),
													},
												);
											}}
										>
											{updateRetry.isPending ? "Saving…" : "Save & publish"}
										</Button>
									</div>
								</>
							) : (
								<p className="text-muted-foreground text-sm">Loading…</p>
							)}
						</section>

						<section className="space-y-3 rounded-md border p-4">
							<h2 className="text-sm font-medium">Email delivery</h2>
							<p className="text-muted-foreground text-sm">
								Choose which platform-enabled mail transport to use for member
								invites. Credentials are configured by the deployment.
							</p>
							{enabled.length === 0 ? (
								<p className="text-muted-foreground text-sm">
									No mail providers are enabled. Invites will be logged on the
									server until SMTP or Resend is configured.
								</p>
							) : (
								<div className="flex flex-col gap-3 sm:flex-row sm:items-end">
									<div className="space-y-1 sm:min-w-56">
										<Label>Provider</Label>
										<Select
											value={selected}
											onValueChange={(value) => {
												if (!value) return;
												updateMail.mutate(
													{ orgId, provider: value },
													{
														onSuccess: () => {
															void qc.invalidateQueries({
																queryKey: ["organizations", orgId, "mail"],
															});
															toast.success("Mail provider updated");
														},
														onError: (err) =>
															toast.error(
																err.message || "Failed to update mail provider",
															),
													},
												);
											}}
										>
											<SelectTrigger className="w-full">
												<SelectValue />
											</SelectTrigger>
											<SelectContent>
												{enabled.map((p) => (
													<SelectItem key={p} value={p}>
														{p}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
									</div>
									<Button
										variant="outline"
										disabled={testMail.isPending}
										onClick={() =>
											testMail.mutate(orgId, {
												onSuccess: (res) =>
													toast.success(`Test email sent to ${res.to}`),
												onError: (err) =>
													toast.error(err.message || "Test email failed"),
											})
										}
									>
										{testMail.isPending ? "Sending…" : "Send test email"}
									</Button>
								</div>
							)}
							{mail.data?.from ? (
								<p className="text-muted-foreground text-xs">
									From: {mail.data.from}
								</p>
							) : null}
						</section>
					</div>

					<section className="space-y-3 rounded-md border p-4">
						<div className="flex items-center justify-between gap-4">
							<div>
								<h2 className="text-sm font-medium">Object storage</h2>
								<p className="text-muted-foreground text-sm">
									Optionally persist generated images to an S3-compatible store.
									When disabled, image responses pass through unchanged.
								</p>
							</div>
							{storeLoaded ? (
								<Switch
									checked={storeDraft.enabled}
									onCheckedChange={(checked) =>
										setStoreDraft((d) => ({ ...d, enabled: checked }))
									}
								/>
							) : null}
						</div>
						{storeLoaded ? (
							<>
								<div className="grid gap-3 sm:grid-cols-2">
									<div className="grid gap-1">
										<Label htmlFor="os-endpoint">Endpoint</Label>
										<Input
											id="os-endpoint"
											value={storeDraft.endpoint ?? ""}
											onChange={(e) =>
												setStoreDraft((d) => ({
													...d,
													endpoint: e.target.value,
												}))
											}
											placeholder="localhost:9000"
											disabled={!storeDraft.enabled}
										/>
									</div>
									<div className="grid gap-1">
										<Label htmlFor="os-bucket">Bucket</Label>
										<Input
											id="os-bucket"
											value={storeDraft.bucket ?? ""}
											onChange={(e) =>
												setStoreDraft((d) => ({ ...d, bucket: e.target.value }))
											}
											placeholder="afi-assets"
											disabled={!storeDraft.enabled}
										/>
									</div>
									<div className="grid gap-1">
										<Label htmlFor="os-region">Region</Label>
										<Input
											id="os-region"
											value={storeDraft.region ?? ""}
											onChange={(e) =>
												setStoreDraft((d) => ({ ...d, region: e.target.value }))
											}
											placeholder="us-east-1"
											disabled={!storeDraft.enabled}
										/>
									</div>
									<div className="grid gap-1">
										<Label htmlFor="os-ttl">Presign TTL (seconds)</Label>
										<Input
											id="os-ttl"
											type="number"
											min={0}
											value={storeDraft.presign_ttl_seconds ?? 3600}
											onChange={(e) =>
												setStoreDraft((d) => ({
													...d,
													presign_ttl_seconds: Number(e.target.value) || 0,
												}))
											}
											disabled={!storeDraft.enabled}
										/>
									</div>
									<div className="grid gap-1">
										<Label htmlFor="os-cred">Credential ID</Label>
										<Input
											id="os-cred"
											value={storeDraft.credential_id ?? ""}
											onChange={(e) =>
												setStoreDraft((d) => ({
													...d,
													credential_id: e.target.value,
												}))
											}
											placeholder="cred_… (JSON access_key/secret_key)"
											disabled={!storeDraft.enabled}
										/>
									</div>
									<div className="grid gap-1 sm:col-span-2 sm:grid-cols-2 sm:gap-3">
										<div className="grid gap-1">
											<Label htmlFor="os-ak">Access key env</Label>
											<Input
												id="os-ak"
												value={storeDraft.access_key_env ?? ""}
												onChange={(e) =>
													setStoreDraft((d) => ({
														...d,
														access_key_env: e.target.value,
													}))
												}
												placeholder="MINIO_ACCESS_KEY"
												disabled={!storeDraft.enabled}
											/>
										</div>
										<div className="grid gap-1">
											<Label htmlFor="os-sk">Secret key env</Label>
											<Input
												id="os-sk"
												value={storeDraft.secret_key_env ?? ""}
												onChange={(e) =>
													setStoreDraft((d) => ({
														...d,
														secret_key_env: e.target.value,
													}))
												}
												placeholder="MINIO_SECRET_KEY"
												disabled={!storeDraft.enabled}
											/>
										</div>
									</div>
								</div>
								<div className="flex flex-wrap items-center gap-4 text-sm">
									<label className="flex items-center gap-2">
										<Switch
											checked={!!storeDraft.use_ssl}
											onCheckedChange={(checked) =>
												setStoreDraft((d) => ({ ...d, use_ssl: checked }))
											}
											disabled={!storeDraft.enabled}
										/>
										Use SSL
									</label>
									<label className="flex items-center gap-2">
										<Switch
											checked={!!storeDraft.path_style}
											onCheckedChange={(checked) =>
												setStoreDraft((d) => ({ ...d, path_style: checked }))
											}
											disabled={!storeDraft.enabled}
										/>
										Path-style addressing
									</label>
								</div>
								<p className="text-muted-foreground text-xs">
									Provide either a credential ID (secret plaintext must be JSON{" "}
									<code>
										{`{"access_key":"…","secret_key":"…"}`}
									</code>
									) or both access/secret env vars on the gateway process.
								</p>
								<div className="flex justify-end">
									<Button
										disabled={updateObjectStore.isPending}
										onClick={() => {
											const payload: ObjectStoreConfig | null = storeDraft.enabled
												? {
														...storeDraft,
														endpoint: storeDraft.endpoint?.trim() || undefined,
														bucket: storeDraft.bucket?.trim() || undefined,
														region: storeDraft.region?.trim() || undefined,
														credential_id:
															storeDraft.credential_id?.trim() || undefined,
														access_key_env:
															storeDraft.access_key_env?.trim() || undefined,
														secret_key_env:
															storeDraft.secret_key_env?.trim() || undefined,
													}
												: { enabled: false };
											updateObjectStore.mutate(
												{ orgId, object_store: payload },
												{
													onSuccess: (res) => {
														setStoreDraft(
															res.object_store ?? emptyObjectStore(),
														);
														void qc.invalidateQueries({
															queryKey: [
																"organizations",
																orgId,
																"object-store",
															],
														});
														toast.success("Object store saved & published");
													},
													onError: (e) =>
														toast.error(
															e instanceof Error
																? e.message
																: "Failed to save object store",
														),
												},
											);
										}}
									>
										{updateObjectStore.isPending
											? "Saving…"
											: "Save & publish"}
									</Button>
								</div>
							</>
						) : (
							<p className="text-muted-foreground text-sm">Loading…</p>
						)}
					</section>
				</>
			) : null}

			<section className="space-y-3 rounded-md border p-4">
				<h2 className="text-sm font-medium">Related</h2>
				<p className="text-muted-foreground text-sm">
					Invite members and manage roles on Users. Configure usage limits on
					Quotas. Per-route retry overrides live on Routing.
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
					<Button
						variant="outline"
						nativeButton={false}
						render={<Link to="/app/routing" />}
					>
						Manage routes
					</Button>
				</div>
			</section>
		</PageBody>
	);
}
