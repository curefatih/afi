import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { providersQueryOptions } from "#/api/provider";
import { quotasQueryOptions } from "#/api/quota";
import { regionOverlayQueryOptions } from "#/api/regions";
import { routesQueryOptions } from "#/api/routing";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { JsonCodeEditor } from "#/components/ui/json-code-editor";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "#/components/ui/tabs";

function rowKey(): string {
	return typeof crypto !== "undefined" && crypto.randomUUID
		? crypto.randomUUID()
		: Math.random().toString(36).substring(2, 15);
}

export type OverlayProviderRow = {
	key: string;
	id: string;
	name: string;
	type: string;
	base_url: string;
};

export type OverlayRouteRow = {
	key: string;
	model: string;
	provider_id: string;
	target_model: string;
};

export type OverlayQuotaRow = {
	key: string;
	id: string;
	scope_type: string;
	scope_id: string;
	metric: string;
	limit_value: number;
	window: string;
};

export type OverlayDraft = {
	providers: OverlayProviderRow[];
	routes: OverlayRouteRow[];
	quotas: OverlayQuotaRow[];
	advanced: Record<string, unknown>;
};

const PROVIDER_TYPES = [
	"openai",
	"anthropic",
	"gemini",
	"azure_openai",
	"bedrock",
	"ollama",
	"echo",
] as const;

const SCOPE_TYPES = [
	"organization",
	"team",
	"project",
	"user",
	"api_key",
] as const;

const METRICS = ["requests", "tokens", "cost"] as const;
const WINDOWS = ["total", "minute", "hour", "day"] as const;

function emptyDraft(_orgId: string): OverlayDraft {
	return {
		providers: [],
		routes: [],
		quotas: [],
		advanced: {},
	};
}

function parsePayload(
	payload: Record<string, unknown>,
	orgId: string,
): OverlayDraft {
	const providers = Array.isArray(payload.providers)
		? (payload.providers as OverlayProviderRow[]).map((p) => ({
				key: rowKey(),
				id: String(p.id ?? ""),
				name: String(p.name ?? ""),
				type: String(p.type ?? "openai"),
				base_url: String(p.base_url ?? ""),
			}))
		: [];
	const routes = Array.isArray(payload.routes)
		? (payload.routes as OverlayRouteRow[]).map((r) => ({
				key: rowKey(),
				model: String(r.model ?? ""),
				provider_id: String(r.provider_id ?? ""),
				target_model: String(r.target_model ?? ""),
			}))
		: [];
	const quotas = Array.isArray(payload.quotas)
		? (payload.quotas as OverlayQuotaRow[]).map((q, i) => ({
				key: rowKey(),
				id: String(q.id ?? `quota_${i + 1}`),
				scope_type: String(q.scope_type ?? "organization"),
				scope_id: String(q.scope_id ?? orgId),
				metric: String(q.metric ?? "requests"),
				limit_value: Number(q.limit_value ?? 0),
				window: String(q.window ?? "total"),
			}))
		: [];
	const advanced: Record<string, unknown> = {};
	for (const key of [
		"policies",
		"wasm_hooks",
		"mcp_backends",
		"a2a_agents",
		"credentials",
		"assignments",
		"default_retry",
		"object_store",
	]) {
		if (payload[key] !== undefined) {
			advanced[key] = payload[key];
		}
	}
	return { providers, routes, quotas, advanced };
}

function draftToPayload(draft: OverlayDraft): Record<string, unknown> {
	const payload: Record<string, unknown> = {
		providers: draft.providers
			.filter((p) => p.id.trim())
			.map(({ key: _key, ...p }) => p),
		routes: draft.routes
			.filter((r) => r.model.trim() && r.provider_id.trim())
			.map(({ key: _key, ...r }) => r),
		quotas: draft.quotas
			.filter((q) => q.scope_id.trim() && q.limit_value > 0)
			.map(({ key: _key, ...q }) => q),
		...draft.advanced,
	};
	return payload;
}

export function RegionOverlaySheet({
	regionId,
	orgId,
	orgLabel,
	onClose,
	onSave,
	onInherit,
	saving,
	inheriting,
}: {
	regionId: string;
	orgId: string;
	orgLabel: string;
	onClose: () => void;
	onSave: (payload: Record<string, unknown>) => void;
	onInherit: () => void;
	saving: boolean;
	inheriting: boolean;
}) {
	const overlayQ = useQuery(regionOverlayQueryOptions(regionId, orgId));
	const providersQ = useQuery(providersQueryOptions(orgId));
	const routesQ = useQuery(routesQueryOptions(orgId));
	const quotasQ = useQuery(quotasQueryOptions(orgId));
	const [draft, setDraft] = useState<OverlayDraft>(() => emptyDraft(orgId));
	const [advancedText, setAdvancedText] = useState("{}");
	const hasOverlay = overlayQ.isSuccess && !!overlayQ.data;

	useEffect(() => {
		if (overlayQ.data?.payload) {
			const parsed = parsePayload(
				overlayQ.data.payload as Record<string, unknown>,
				orgId,
			);
			setDraft(parsed);
			setAdvancedText(JSON.stringify(parsed.advanced, null, 2));
		} else if (overlayQ.isError || overlayQ.isSuccess) {
			setDraft(emptyDraft(orgId));
			setAdvancedText("{}");
		}
	}, [overlayQ.data, overlayQ.isError, overlayQ.isSuccess, orgId]);

	const copyFromBase = () => {
		const providers = (providersQ.data ?? []).map((p) => ({
			key: rowKey(),
			id: p.id,
			name: p.name,
			type: p.type,
			base_url: p.base_url,
		}));
		const routes = (routesQ.data ?? []).map((r) => ({
			key: rowKey(),
			model: r.model,
			provider_id: r.provider_id,
			target_model: r.target_model,
		}));
		const quotas = (quotasQ.data ?? []).map((q) => ({
			key: rowKey(),
			id: q.id,
			scope_type: q.scope_type,
			scope_id: q.scope_id,
			metric: q.metric,
			limit_value: q.limit_value,
			window: q.window,
		}));
		setDraft((prev) => ({ ...prev, providers, routes, quotas }));
		toast.success("Copied base providers, routes, and quotas into draft");
	};

	const save = () => {
		let advanced: Record<string, unknown> = {};
		try {
			advanced = JSON.parse(advancedText) as Record<string, unknown>;
			if (
				advanced === null ||
				Array.isArray(advanced) ||
				typeof advanced !== "object"
			) {
				toast.error("Advanced JSON must be an object");
				return;
			}
		} catch {
			toast.error("Invalid advanced JSON");
			return;
		}
		onSave(draftToPayload({ ...draft, advanced }));
	};

	return (
		<Sheet open onOpenChange={(o) => !o && onClose()}>
			<SheetContent className="w-full overflow-y-auto sm:max-w-4xl data-[side=right]:sm:max-w-4xl data-[side=left]:sm:max-w-4xl">
				<SheetHeader>
					<SheetTitle>Config overlay</SheetTitle>
					<SheetDescription>
						{orgLabel}:{" "}
						{hasOverlay
							? "replace mode — edits fully replace this org’s gateway slice in the region"
							: "no overlay yet — saving creates a full replace; inherit keeps base config"}
					</SheetDescription>
				</SheetHeader>
				<div className="flex flex-1 flex-col gap-4 px-4 pb-4">
					<div className="flex flex-wrap items-center justify-between gap-2">
						<p className="text-muted-foreground text-xs">
							API keys stay global. Presence of an overlay replaces providers,
							routes, quotas (and optional advanced kinds) for this region only.
						</p>
						<Button
							type="button"
							variant="outline"
							size="sm"
							disabled={
								providersQ.isPending || routesQ.isPending || quotasQ.isPending
							}
							onClick={copyFromBase}
						>
							Copy from base
						</Button>
					</div>
					<Tabs defaultValue="routes">
						<TabsList className="w-full">
							<TabsTrigger value="routes">
								Routes ({draft.routes.length})
							</TabsTrigger>
							<TabsTrigger value="providers">
								Providers ({draft.providers.length})
							</TabsTrigger>
							<TabsTrigger value="quotas">
								Quotas ({draft.quotas.length})
							</TabsTrigger>
							<TabsTrigger value="advanced">Advanced</TabsTrigger>
						</TabsList>

						<TabsContent value="routes" className="space-y-3 pt-3">
							{draft.routes.map((row, i) => (
								<div key={row.key} className="grid gap-2 rounded-md border p-3">
									<div className="grid gap-2 sm:grid-cols-3">
										<div className="grid gap-1">
											<Label>Model</Label>
											<Input
												value={row.model}
												onChange={(e) => {
													const routes = [...draft.routes];
													routes[i] = { ...row, model: e.target.value };
													setDraft({ ...draft, routes });
												}}
												placeholder="gpt-4o-mini"
											/>
										</div>
										<div className="grid gap-1">
											<Label>Provider ID</Label>
											<Input
												value={row.provider_id}
												onChange={(e) => {
													const routes = [...draft.routes];
													routes[i] = {
														...row,
														provider_id: e.target.value,
													};
													setDraft({ ...draft, routes });
												}}
												placeholder="prov_openai"
											/>
										</div>
										<div className="grid gap-1">
											<Label>Target model</Label>
											<Input
												value={row.target_model}
												onChange={(e) => {
													const routes = [...draft.routes];
													routes[i] = {
														...row,
														target_model: e.target.value,
													};
													setDraft({ ...draft, routes });
												}}
												placeholder="gpt-4o-mini"
											/>
										</div>
									</div>
									<div className="flex justify-end">
										<Button
											type="button"
											variant="ghost"
											size="sm"
											onClick={() =>
												setDraft({
													...draft,
													routes: draft.routes.filter((_, j) => j !== i),
												})
											}
										>
											Remove
										</Button>
									</div>
								</div>
							))}
							<Button
								type="button"
								variant="outline"
								size="sm"
								onClick={() =>
									setDraft({
										...draft,
										routes: [
											...draft.routes,
											{
												key: rowKey(),
												model: "",
												provider_id: "",
												target_model: "",
											},
										],
									})
								}
							>
								Add route
							</Button>
						</TabsContent>

						<TabsContent value="providers" className="space-y-3 pt-3">
							{draft.providers.map((row, i) => (
								<div key={row.key} className="grid gap-2 rounded-md border p-3">
									<div className="grid gap-2 sm:grid-cols-2">
										<div className="grid gap-1">
											<Label>ID</Label>
											<Input
												value={row.id}
												onChange={(e) => {
													const providers = [...draft.providers];
													providers[i] = { ...row, id: e.target.value };
													setDraft({ ...draft, providers });
												}}
												placeholder="prov_openai"
											/>
										</div>
										<div className="grid gap-1">
											<Label>Name</Label>
											<Input
												value={row.name}
												onChange={(e) => {
													const providers = [...draft.providers];
													providers[i] = { ...row, name: e.target.value };
													setDraft({ ...draft, providers });
												}}
											/>
										</div>
										<div className="grid gap-1">
											<Label>Type</Label>
											<Select
												value={row.type}
												onValueChange={(v) => {
													const providers = [...draft.providers];
													providers[i] = {
														...row,
														type: v ?? "openai",
													};
													setDraft({ ...draft, providers });
												}}
											>
												<SelectTrigger>
													<SelectValue />
												</SelectTrigger>
												<SelectContent>
													{PROVIDER_TYPES.map((t) => (
														<SelectItem key={t} value={t}>
															{t}
														</SelectItem>
													))}
												</SelectContent>
											</Select>
										</div>
										<div className="grid gap-1">
											<Label>Base URL</Label>
											<Input
												value={row.base_url}
												onChange={(e) => {
													const providers = [...draft.providers];
													providers[i] = {
														...row,
														base_url: e.target.value,
													};
													setDraft({ ...draft, providers });
												}}
												placeholder="https://api.openai.com/v1"
											/>
										</div>
									</div>
									<div className="flex justify-end">
										<Button
											type="button"
											variant="ghost"
											size="sm"
											onClick={() =>
												setDraft({
													...draft,
													providers: draft.providers.filter((_, j) => j !== i),
												})
											}
										>
											Remove
										</Button>
									</div>
								</div>
							))}
							<Button
								type="button"
								variant="outline"
								size="sm"
								onClick={() =>
									setDraft({
										...draft,
										providers: [
											...draft.providers,
											{
												key: rowKey(),
												id: "",
												name: "",
												type: "openai",
												base_url: "https://api.openai.com/v1",
											},
										],
									})
								}
							>
								Add provider
							</Button>
						</TabsContent>

						<TabsContent value="quotas" className="space-y-3 pt-3">
							{draft.quotas.map((row, i) => (
								<div key={row.key} className="grid gap-2 rounded-md border p-3">
									<div className="grid gap-2 sm:grid-cols-2">
										<div className="grid gap-1">
											<Label>ID</Label>
											<Input
												value={row.id}
												onChange={(e) => {
													const quotas = [...draft.quotas];
													quotas[i] = { ...row, id: e.target.value };
													setDraft({ ...draft, quotas });
												}}
											/>
										</div>
										<div className="grid gap-1">
											<Label>Scope</Label>
											<Select
												value={row.scope_type}
												onValueChange={(v) => {
													const next = v ?? "organization";
													const quotas = [...draft.quotas];
													quotas[i] = {
														...row,
														scope_type: next,
														scope_id:
															next === "organization" ? orgId : row.scope_id,
													};
													setDraft({ ...draft, quotas });
												}}
											>
												<SelectTrigger>
													<SelectValue />
												</SelectTrigger>
												<SelectContent>
													{SCOPE_TYPES.map((s) => (
														<SelectItem key={s} value={s}>
															{s}
														</SelectItem>
													))}
												</SelectContent>
											</Select>
										</div>
										<div className="grid gap-1">
											<Label>Scope ID</Label>
											<Input
												value={row.scope_id}
												onChange={(e) => {
													const quotas = [...draft.quotas];
													quotas[i] = {
														...row,
														scope_id: e.target.value,
													};
													setDraft({ ...draft, quotas });
												}}
											/>
										</div>
										<div className="grid gap-1">
											<Label>Metric</Label>
											<Select
												value={row.metric}
												onValueChange={(v) => {
													const quotas = [...draft.quotas];
													quotas[i] = {
														...row,
														metric: v ?? "requests",
													};
													setDraft({ ...draft, quotas });
												}}
											>
												<SelectTrigger>
													<SelectValue />
												</SelectTrigger>
												<SelectContent>
													{METRICS.map((m) => (
														<SelectItem key={m} value={m}>
															{m}
														</SelectItem>
													))}
												</SelectContent>
											</Select>
										</div>
										<div className="grid gap-1">
											<Label>Window</Label>
											<Select
												value={row.window}
												onValueChange={(v) => {
													const quotas = [...draft.quotas];
													quotas[i] = {
														...row,
														window: v ?? "total",
													};
													setDraft({ ...draft, quotas });
												}}
											>
												<SelectTrigger>
													<SelectValue />
												</SelectTrigger>
												<SelectContent>
													{WINDOWS.map((w) => (
														<SelectItem key={w} value={w}>
															{w}
														</SelectItem>
													))}
												</SelectContent>
											</Select>
										</div>
										<div className="grid gap-1">
											<Label>Limit</Label>
											<Input
												type="number"
												value={row.limit_value}
												onChange={(e) => {
													const quotas = [...draft.quotas];
													quotas[i] = {
														...row,
														limit_value: Number(e.target.value) || 0,
													};
													setDraft({ ...draft, quotas });
												}}
											/>
										</div>
									</div>
									<div className="flex justify-end">
										<Button
											type="button"
											variant="ghost"
											size="sm"
											onClick={() =>
												setDraft({
													...draft,
													quotas: draft.quotas.filter((_, j) => j !== i),
												})
											}
										>
											Remove
										</Button>
									</div>
								</div>
							))}
							<Button
								type="button"
								variant="outline"
								size="sm"
								onClick={() =>
									setDraft({
										...draft,
										quotas: [
											...draft.quotas,
											{
												key: rowKey(),
												id: `quota_${draft.quotas.length + 1}`,
												scope_type: "organization",
												scope_id: orgId,
												metric: "requests",
												limit_value: 1000,
												window: "total",
											},
										],
									})
								}
							>
								Add quota
							</Button>
						</TabsContent>

						<TabsContent value="advanced" className="space-y-2 pt-3">
							<div className="flex items-center justify-between gap-2">
								<Label htmlFor="overlay-advanced">
									Extra kinds (JSON object)
								</Label>
								<span className="text-[11px] text-muted-foreground">
									Tab indent · Shift+Tab outdent
								</span>
							</div>
							<JsonCodeEditor
								id="overlay-advanced"
								value={advancedText}
								onChange={setAdvancedText}
								minHeight="14rem"
								placeholder="{}"
							/>
							<p className="text-muted-foreground text-xs">
								Optional: policies, wasm_hooks, mcp_backends, a2a_agents,
								credentials, assignments, default_retry, object_store.
							</p>
						</TabsContent>
					</Tabs>

					<SheetFooter className="gap-2 px-0 sm:flex-col sm:space-x-0">
						{hasOverlay ? (
							<Button
								type="button"
								variant="outline"
								disabled={inheriting}
								onClick={onInherit}
							>
								{inheriting ? "Reverting…" : "Inherit base"}
							</Button>
						) : null}
						<Button type="button" variant="outline" onClick={onClose}>
							Cancel
						</Button>
						<Button type="button" disabled={saving} onClick={save}>
							{saving ? "Saving…" : "Save replace overlay"}
						</Button>
					</SheetFooter>
				</div>
			</SheetContent>
		</Sheet>
	);
}
