import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Building2Icon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import {
	orgDefaultRetryQueryOptions,
	orgMailQueryOptions,
	orgMembersQueryOptions,
	testOrgMailMutationOptions,
	updateOrgDefaultRetryMutationOptions,
	updateOrgMailMutationOptions,
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
import { Label } from "#/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/settings/general")({
	...pageTitle("Organization settings"),
	component: RouteComponent,
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
	const updateMail = useMutation(updateOrgMailMutationOptions());
	const testMail = useMutation(testOrgMailMutationOptions());
	const updateRetry = useMutation(updateOrgDefaultRetryMutationOptions());
	const [retryDraft, setRetryDraft] = useState<RetryConfig | null>(null);
	const [retryLoaded, setRetryLoaded] = useState(false);

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	useEffect(() => {
		if (!defaultRetry.isSuccess) return;
		setRetryDraft(defaultRetry.data.retry ?? null);
		setRetryLoaded(true);
	}, [defaultRetry.isSuccess, defaultRetry.data]);

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
				<div className="grid gap-4 lg:grid-cols-2">
					<section className="space-y-3 rounded-md border p-4">
						<h2 className="text-sm font-medium">Default retry</h2>
						<p className="text-muted-foreground text-sm">
							Used for routes that do not set their own retry. Route-level retry
							always wins when configured.
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
