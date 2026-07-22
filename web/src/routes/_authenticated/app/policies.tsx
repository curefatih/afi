import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { PlusIcon, ShieldCheckIcon, Trash2Icon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { credentialsQueryOptions } from "#/api/credentials";
import { orgMembersQueryOptions } from "#/api/organization";
import {
	createPolicyMutationOptions,
	deletePolicyMutationOptions,
	type PolicyActionConfig,
	type PolicyActionType,
	type PolicyThen,
	policiesQueryOptions,
	policyActions,
	type RequestPolicy,
	reorderPoliciesMutationOptions,
	updatePolicyMutationOptions,
} from "#/api/policies";
import { InfoAlert } from "#/components/info-alert";
import { PageBody, PageHeader } from "#/components/page-header";
import { CelExpressionEditor } from "#/components/policies/cel-expression-editor";
import { SortablePolicyTable } from "#/components/policies/sortable-policy-table";
import { QueryGate } from "#/components/query-state";
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
import { Switch } from "#/components/ui/switch";
import { CEL_EXAMPLES, CEL_VARIABLES } from "#/lib/cel-policy-catalog";
import { pageTitle } from "#/lib/page-meta";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/policies")({
	...pageTitle("Policies"),
	component: RouteComponent,
});

const ACTIONS: Array<{
	value: PolicyActionType;
	label: string;
	hint: string;
}> = [
	{
		value: "deny",
		label: "Deny",
		hint: "Reject the request with HTTP 403. Stops further Then steps and lower-priority policies.",
	},
	{
		value: "allow",
		label: "Allow",
		hint: "Short-circuit: allow immediately (skips remaining Then steps and lower-priority rules).",
	},
	{
		value: "set_header",
		label: "Update header",
		hint: "Set an outbound header on the upstream provider request, then continue.",
	},
	{
		value: "use_credential",
		label: "Use credential",
		hint: "Switch to a named secret for this request, then continue.",
	},
];

type ThenForm = {
	id: string;
	action: PolicyActionType;
	header: string;
	headerValue: string;
	headerValueMode: "static" | "expr";
	headerValueExpr: string;
	credentialMode: "static" | "expr";
	credentialName: string;
	credentialNameExpr: string;
};

type FormState = {
	name: string;
	expression: string;
	thens: ThenForm[];
	priority: string;
	enabled: boolean;
};

function newThenId(): string {
	return typeof crypto !== "undefined" && crypto.randomUUID
		? crypto.randomUUID()
		: Math.random().toString(36).substring(2, 15);
}

const defaultThen = (): ThenForm => ({
	id: newThenId(),
	action: "deny",
	header: "",
	headerValue: "",
	headerValueMode: "static",
	headerValueExpr: 'request.headers["x-tenant-id"]',
	credentialMode: "static",
	credentialName: "",
	credentialNameExpr: 'request.headers["x-tenant-id"]',
});

const defaultForm = (): FormState => ({
	name: "",
	expression: 'request.model == "blocked-model"',
	thens: [defaultThen()],
	priority: "100",
	enabled: true,
});

function thenFromAction(a: PolicyThen): ThenForm {
	const cfg = a.config ?? {};
	const credExpr = Boolean(cfg.credential_name_expr);
	const valueExpr = Boolean(cfg.value_expr);
	return {
		id: newThenId(),
		action: a.type || "deny",
		header: cfg.header ?? "",
		headerValue: cfg.value ?? "",
		headerValueMode: valueExpr ? "expr" : "static",
		headerValueExpr: cfg.value_expr ?? 'request.headers["x-tenant-id"]',
		credentialMode: credExpr ? "expr" : "static",
		credentialName: cfg.credential_name ?? "",
		credentialNameExpr:
			cfg.credential_name_expr ?? 'request.headers["x-tenant-id"]',
	};
}

function buildActionConfig(t: ThenForm): PolicyActionConfig {
	switch (t.action) {
		case "set_header":
			if (t.headerValueMode === "expr") {
				return {
					header: t.header.trim(),
					value_expr: t.headerValueExpr.trim(),
				};
			}
			return { header: t.header.trim(), value: t.headerValue };
		case "use_credential":
			if (t.credentialMode === "expr") {
				return { credential_name_expr: t.credentialNameExpr.trim() };
			}
			return { credential_name: t.credentialName.trim() };
		default:
			return {};
	}
}

function buildActions(f: FormState): PolicyThen[] {
	return f.thens.map((t) => ({
		type: t.action,
		config: buildActionConfig(t),
	}));
}

function formFromPolicy(p: RequestPolicy): FormState {
	const actions = policyActions(p);
	return {
		name: p.name,
		expression: p.expression,
		thens: actions.map(thenFromAction),
		priority: String(p.priority),
		enabled: p.enabled,
	};
}

function thenIncomplete(t: ThenForm): boolean {
	if (t.action === "set_header") {
		return (
			!t.header.trim() ||
			(t.headerValueMode === "expr" && !t.headerValueExpr.trim())
		);
	}
	if (t.action === "use_credential") {
		return (
			(t.credentialMode === "static" && !t.credentialName.trim()) ||
			(t.credentialMode === "expr" && !t.credentialNameExpr.trim())
		);
	}
	return false;
}

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const user = useAuthUser();
	const qc = useQueryClient();
	const policies = useQuery(policiesQueryOptions(orgId));
	const members = useQuery(orgMembersQueryOptions(orgId));
	const credentials = useQuery(credentialsQueryOptions(orgId));
	const [createOpen, setCreateOpen] = useState(false);
	const [edit, setEdit] = useState<RequestPolicy | null>(null);
	const [createForm, setCreateForm] = useState<FormState>(defaultForm);
	const [editForm, setEditForm] = useState<FormState>(defaultForm);
	const [error, setError] = useState<string | null>(null);
	const [editError, setEditError] = useState<string | null>(null);

	const credentialNames = useMemo(() => {
		return (credentials.data ?? [])
			.map((c) => c.name)
			.filter(Boolean)
			.sort((a, b) => a.localeCompare(b));
	}, [credentials.data]);

	const isOrgAdmin = useMemo(() => {
		const me = (members.data ?? []).find((m) => m.user_id === user?.id);
		return me?.role === "owner" || me?.role === "admin";
	}, [members.data, user?.id]);

	const invalidate = () =>
		void qc.invalidateQueries({
			queryKey: ["organizations", orgId, "policies"],
		});

	const create = useMutation({
		...createPolicyMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("Policy created");
			setCreateOpen(false);
			setCreateForm(defaultForm());
			setError(null);
		},
	});
	const update = useMutation({
		...updatePolicyMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("Policy updated");
			setEdit(null);
		},
	});
	const del = useMutation({
		...deletePolicyMutationOptions(),
		onSuccess: () => {
			invalidate();
			toast.success("Policy deleted");
		},
	});
	const reorder = useMutation({
		...reorderPoliciesMutationOptions(),
		onMutate: async (vars) => {
			const queryKey = ["organizations", orgId, "policies"] as const;
			await qc.cancelQueries({ queryKey });
			const previous = qc.getQueryData<RequestPolicy[]>(queryKey);
			if (previous) {
				const byId = new Map(previous.map((p) => [p.id, p]));
				qc.setQueryData<RequestPolicy[]>(
					queryKey,
					vars.policies.flatMap((p) => {
						const cur = byId.get(p.id);
						return cur ? [{ ...cur, priority: p.priority }] : [];
					}),
				);
			}
			return { previous };
		},
		onError: (err, _vars, ctx) => {
			if (ctx?.previous) {
				qc.setQueryData(["organizations", orgId, "policies"], ctx.previous);
			}
			toast.error(
				err instanceof Error ? err.message : "Failed to reorder policies",
			);
		},
		onSuccess: (list) => {
			qc.setQueryData(["organizations", orgId, "policies"], list);
			toast.success("Policy order updated");
		},
		onSettled: invalidate,
	});

	const list = useMemo(() => {
		const rows = [...(policies.data ?? [])];
		rows.sort((a, b) => {
			if (a.priority !== b.priority) return b.priority - a.priority;
			return a.name.localeCompare(b.name);
		});
		return rows;
	}, [policies.data]);

	const handleReorder = (next: RequestPolicy[]) => {
		const sameOrder =
			next.length === list.length &&
			next.every(
				(p, i) => p.id === list[i]?.id && p.priority === list[i]?.priority,
			);
		if (sameOrder) return;
		reorder.mutate({
			orgId,
			policies: next.map((p) => ({ id: p.id, priority: p.priority })),
		});
	};

	const openEdit = (p: RequestPolicy) => {
		setEdit(p);
		setEditForm(formFromPolicy(p));
		setEditError(null);
	};

	return (
		<PageBody>
			<PageHeader
				title="Policies"
				description="When/then rules for gateway traffic."
				info="When the CEL expression is true, Then actions run in order. Higher priority policies run first. Deny/allow stop evaluation; header and credential actions continue."
				actions={
					isOrgAdmin ? (
						<Button onClick={() => setCreateOpen(true)} disabled={!orgId}>
							<PlusIcon />
							Add policy
						</Button>
					) : null
				}
			/>

			<div className="rounded-lg border bg-muted/20 p-4 text-sm space-y-3">
				<div>
					<p className="font-medium">Quick start</p>
					<p className="text-muted-foreground text-xs mt-1 leading-relaxed">
						Write a When condition with{" "}
						<code className="text-foreground">request.*</code> /{" "}
						<code className="text-foreground">key.*</code>, then add one or more
						Then steps: deny, allow, update header, or use credential.
					</p>
				</div>
				<div className="flex flex-wrap gap-1.5">
					{CEL_VARIABLES.filter((v) => v.type === "field").map((v) => (
						<code
							key={v.label}
							className="rounded-md border bg-background px-1.5 py-0.5 font-mono text-[11px]"
							title={v.detail}
						>
							{v.label}
						</code>
					))}
				</div>
				<div className="grid gap-2 sm:grid-cols-2">
					{CEL_EXAMPLES.slice(0, 4).map((ex) => (
						<div
							key={ex.title}
							className="rounded-md border bg-background/80 px-3 py-2"
						>
							<p className="text-xs font-medium">{ex.title}</p>
							<code className="mt-1 block font-mono text-[11px] text-muted-foreground truncate">
								{ex.expression}
							</code>
						</div>
					))}
				</div>
			</div>

			<QueryGate
				isPending={policies.isPending || members.isPending}
				isError={policies.isError}
				error={policies.error}
				onRetry={() => policies.refetch()}
			>
				{list.length === 0 ? (
					<Empty className="border min-h-64">
						<EmptyHeader>
							<EmptyMedia variant="icon">
								<ShieldCheckIcon />
							</EmptyMedia>
							<EmptyTitle>No policies yet</EmptyTitle>
							<EmptyDescription>
								Traffic is allowed until you add a when/then rule.
								{!isOrgAdmin
									? " Only organization owners and admins can create policies."
									: ""}
							</EmptyDescription>
						</EmptyHeader>
						{isOrgAdmin ? (
							<EmptyContent>
								<Button onClick={() => setCreateOpen(true)} disabled={!orgId}>
									<PlusIcon />
									Add policy
								</Button>
							</EmptyContent>
						) : null}
					</Empty>
				) : (
					<>
						{!isOrgAdmin ? (
							<p className="text-muted-foreground text-sm">
								Only organization owners and admins can create or edit policies.
							</p>
						) : null}
						<SortablePolicyTable
							policies={list}
							canEdit={isOrgAdmin}
							disabled={reorder.isPending}
							onReorder={handleReorder}
							onEdit={openEdit}
							onDelete={(id) => del.mutate(id)}
							deletePending={del.isPending}
						/>
					</>
				)}
			</QueryGate>

			<PolicySheet
				open={createOpen}
				onOpenChange={setCreateOpen}
				title="Add policy"
				description="When the expression matches, run Then actions in order."
				form={createForm}
				setForm={setCreateForm}
				credentialNames={credentialNames}
				error={error}
				pending={create.isPending}
				submitLabel={create.isPending ? "Creating…" : "Create & publish"}
				onSubmit={() => {
					if (!orgId) return;
					setError(null);
					create.mutate(
						{
							orgId,
							name: createForm.name,
							expression: createForm.expression,
							actions: buildActions(createForm),
							priority: Number(createForm.priority) || 100,
						},
						{
							onError: (err) =>
								setError(err instanceof Error ? err.message : "Create failed"),
						},
					);
				}}
			/>

			<PolicySheet
				open={!!edit}
				onOpenChange={(o) => {
					if (!o) setEdit(null);
				}}
				title="Edit policy"
				description="Update when/then, priority, or enable/disable."
				form={editForm}
				setForm={setEditForm}
				credentialNames={credentialNames}
				error={editError}
				pending={update.isPending}
				submitLabel={update.isPending ? "Saving…" : "Save & publish"}
				showEnabled
				onSubmit={() => {
					if (!edit) return;
					setEditError(null);
					update.mutate(
						{
							policyId: edit.id,
							name: editForm.name,
							expression: editForm.expression,
							actions: buildActions(editForm),
							priority:
								editForm.priority.trim() === "" ||
								Number.isNaN(Number(editForm.priority))
									? 100
									: Number(editForm.priority),
							enabled: editForm.enabled,
						},
						{
							onError: (err) =>
								setEditError(
									err instanceof Error ? err.message : "Update failed",
								),
						},
					);
				}}
			/>
		</PageBody>
	);
}

function PolicySheet({
	open,
	onOpenChange,
	title,
	description,
	form,
	setForm,
	credentialNames,
	error,
	pending,
	submitLabel,
	onSubmit,
	showEnabled,
}: {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	title: string;
	description: string;
	form: FormState;
	setForm: (f: FormState | ((prev: FormState) => FormState)) => void;
	credentialNames: string[];
	error: string | null;
	pending: boolean;
	submitLabel: string;
	onSubmit: () => void;
	showEnabled?: boolean;
}) {
	const updateThen = (index: number, patch: Partial<ThenForm>) => {
		setForm((prev) => ({
			...prev,
			thens: prev.thens.map((t, i) => (i === index ? { ...t, ...patch } : t)),
		}));
	};

	const removeThen = (index: number) => {
		setForm((prev) => ({
			...prev,
			thens:
				prev.thens.length <= 1
					? prev.thens
					: prev.thens.filter((_, i) => i !== index),
		}));
	};

	const addThen = () => {
		setForm((prev) => ({
			...prev,
			thens: [
				...prev.thens,
				{
					...defaultThen(),
					action: "use_credential",
				},
			],
		}));
	};

	const submitDisabled =
		pending ||
		!form.name.trim() ||
		form.thens.length === 0 ||
		form.thens.some(thenIncomplete);

	return (
		<Sheet open={open} onOpenChange={onOpenChange}>
			<SheetContent className="w-full overflow-y-auto sm:max-w-2xl data-[side=right]:sm:max-w-2xl data-[side=left]:sm:max-w-2xl">
				<SheetHeader>
					<SheetTitle>{title}</SheetTitle>
					<SheetDescription>{description}</SheetDescription>
					<InfoAlert>
						Matching rules run by priority. Within a match, Then steps run in
						order. Deny and allow stop; update header and use credential
						continue.
					</InfoAlert>
				</SheetHeader>
				<form
					className="flex flex-1 flex-col gap-4 px-4 pb-4"
					onSubmit={(e) => {
						e.preventDefault();
						onSubmit();
					}}
				>
					<div className="space-y-1">
						<Label htmlFor="pol-name">Name</Label>
						<Input
							id="pol-name"
							value={form.name}
							placeholder="block-risky-model"
							onChange={(e) => setForm({ ...form, name: e.target.value })}
							required
						/>
					</div>
					<div className="space-y-1">
						<Label htmlFor="pol-priority">Priority</Label>
						<Input
							id="pol-priority"
							type="number"
							value={form.priority}
							onChange={(e) => setForm({ ...form, priority: e.target.value })}
						/>
						<p className="text-[11px] text-muted-foreground">
							Higher priority runs first.
						</p>
					</div>

					<section className="rounded-lg border">
						<div className="space-y-3 p-3">
							<div className="flex items-baseline gap-2">
								<span className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
									When
								</span>
								<p className="text-[11px] text-muted-foreground">
									CEL expression must be true for Then steps to run.
								</p>
							</div>
							<CelExpressionEditor
								id="pol-expr"
								value={form.expression}
								onChange={(expression) => setForm({ ...form, expression })}
							/>
						</div>

						{form.thens.map((then, index) => (
							<ThenBlock
								key={then.id}
								index={index}
								thenCount={form.thens.length}
								then={then}
								canRemove={form.thens.length > 1}
								credentialNames={credentialNames}
								onChange={(patch) => updateThen(index, patch)}
								onRemove={() => removeThen(index)}
							/>
						))}

						<div className="border-t px-3 py-2">
							<Button
								type="button"
								variant="outline"
								size="sm"
								onClick={addThen}
							>
								<PlusIcon />
								Add Then
							</Button>
						</div>
					</section>

					{showEnabled ? (
						<div className="flex items-center justify-between gap-2">
							<div>
								<Label htmlFor="edit-pol-enabled">Enabled</Label>
								<p className="text-[11px] text-muted-foreground">
									Disabled policies are skipped.
								</p>
							</div>
							<Switch
								id="edit-pol-enabled"
								checked={form.enabled}
								onCheckedChange={(enabled) => setForm({ ...form, enabled })}
							/>
						</div>
					) : null}

					{error ? <p className="text-destructive text-xs">{error}</p> : null}
					<SheetFooter>
						<Button
							type="button"
							variant="outline"
							onClick={() => onOpenChange(false)}
						>
							Cancel
						</Button>
						<Button type="submit" disabled={submitDisabled}>
							{submitLabel}
						</Button>
					</SheetFooter>
				</form>
			</SheetContent>
		</Sheet>
	);
}

function ThenBlock({
	index,
	thenCount,
	then,
	canRemove,
	credentialNames,
	onChange,
	onRemove,
}: {
	index: number;
	thenCount: number;
	then: ThenForm;
	canRemove: boolean;
	credentialNames: string[];
	onChange: (patch: Partial<ThenForm>) => void;
	onRemove: () => void;
}) {
	const actionMeta = ACTIONS.find((a) => a.value === then.action);
	const idPrefix = `pol-then-${index}`;

	return (
		<div className="border-t">
			<div className="space-y-3 border-l-2 border-muted-foreground/25 py-3 pr-3 pl-4 ml-3">
				<div className="flex items-center justify-between gap-2">
					<div className="flex items-baseline gap-2">
						<span className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
							Then{thenCount > 1 ? ` ${index + 1}` : ""}
						</span>
						<p className="text-[11px] text-muted-foreground">
							{index === 0
								? "First action when When matches."
								: "Runs after previous Then steps (unless stopped)."}
						</p>
					</div>
					{canRemove ? (
						<Button
							type="button"
							variant="ghost"
							size="icon-xs"
							aria-label={`Remove Then ${index + 1}`}
							onClick={onRemove}
						>
							<Trash2Icon />
						</Button>
					) : null}
				</div>

				<div className="space-y-1">
					<Label htmlFor={`${idPrefix}-action`}>Action</Label>
					<Select
						value={then.action}
						onValueChange={(v) => onChange({ action: v as PolicyActionType })}
					>
						<SelectTrigger id={`${idPrefix}-action`}>
							<SelectValue />
						</SelectTrigger>
						<SelectContent>
							{ACTIONS.map((a) => (
								<SelectItem key={a.value} value={a.value}>
									{a.label}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
					{actionMeta ? (
						<p className="text-[11px] text-muted-foreground">
							{actionMeta.hint}
						</p>
					) : null}
				</div>

				{then.action === "set_header" ? (
					<div className="space-y-3 border-l-2 border-muted-foreground/20 py-1 pl-3">
						<div className="grid gap-3 sm:grid-cols-2">
							<div className="space-y-1">
								<Label htmlFor={`${idPrefix}-hdr`}>Header</Label>
								<Input
									id={`${idPrefix}-hdr`}
									value={then.header}
									placeholder="X-Partner-Id"
									onChange={(e) => onChange({ header: e.target.value })}
									required
								/>
							</div>
							<div className="space-y-1">
								<Label htmlFor={`${idPrefix}-hdr-mode`}>Value source</Label>
								<Select
									value={then.headerValueMode}
									onValueChange={(v) =>
										onChange({
											headerValueMode: v as "static" | "expr",
										})
									}
								>
									<SelectTrigger id={`${idPrefix}-hdr-mode`}>
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="static">Static value</SelectItem>
										<SelectItem value="expr">From CEL expression</SelectItem>
									</SelectContent>
								</Select>
							</div>
						</div>
						{then.headerValueMode === "static" ? (
							<div className="space-y-1">
								<Label htmlFor={`${idPrefix}-hdr-val`}>Value</Label>
								<Input
									id={`${idPrefix}-hdr-val`}
									value={then.headerValue}
									placeholder="acme"
									onChange={(e) => onChange({ headerValue: e.target.value })}
								/>
							</div>
						) : (
							<div className="space-y-1 border-l-2 border-muted-foreground/15 pl-3">
								<Label htmlFor={`${idPrefix}-hdr-expr`}>Value expression</Label>
								<Input
									id={`${idPrefix}-hdr-expr`}
									value={then.headerValueExpr}
									placeholder={'request.headers["x-tenant-id"]'}
									onChange={(e) =>
										onChange({ headerValueExpr: e.target.value })
									}
									className="font-mono text-sm"
									required
								/>
								<p className="text-[11px] text-muted-foreground">
									Must evaluate to a string (same vars as When).
								</p>
							</div>
						)}
					</div>
				) : null}

				{then.action === "use_credential" ? (
					<div className="space-y-3 border-l-2 border-muted-foreground/20 py-1 pl-3">
						<div className="space-y-1">
							<Label htmlFor={`${idPrefix}-cred-mode`}>Credential source</Label>
							<Select
								value={then.credentialMode}
								onValueChange={(v) =>
									onChange({
										credentialMode: v as "static" | "expr",
									})
								}
							>
								<SelectTrigger id={`${idPrefix}-cred-mode`}>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="static">Named secret</SelectItem>
									<SelectItem value="expr">From CEL expression</SelectItem>
								</SelectContent>
							</Select>
							<p className="text-[11px] text-muted-foreground">
								Expression mode uses the resolved string as the credential name
								(e.g. header value → secret name).
							</p>
						</div>
						{then.credentialMode === "static" ? (
							<div className="space-y-1">
								<Label htmlFor={`${idPrefix}-cred`}>Credential</Label>
								<Select
									value={then.credentialName || undefined}
									onValueChange={(v) => onChange({ credentialName: v ?? "" })}
								>
									<SelectTrigger id={`${idPrefix}-cred`}>
										<SelectValue placeholder="Select a secret" />
									</SelectTrigger>
									<SelectContent>
										{credentialNames.map((n) => (
											<SelectItem key={n} value={n}>
												{n}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
						) : (
							<div className="space-y-1 border-l-2 border-muted-foreground/15 pl-3">
								<Label htmlFor={`${idPrefix}-cred-expr`}>Name expression</Label>
								<Input
									id={`${idPrefix}-cred-expr`}
									value={then.credentialNameExpr}
									placeholder={'request.headers["x-tenant-id"]'}
									onChange={(e) =>
										onChange({ credentialNameExpr: e.target.value })
									}
									className="font-mono text-sm"
									required
								/>
								<p className="text-[11px] text-muted-foreground">
									Example: When{" "}
									<code>{'("x-tenant-id" in request.headers)'}</code>, Then use{" "}
									<code>{'request.headers["x-tenant-id"]'}</code>.
								</p>
							</div>
						)}
					</div>
				) : null}
			</div>
		</div>
	);
}
