import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Loader2Icon } from "lucide-react";
import { useEffect, useId, useMemo, useRef, useState } from "react";
import { a2aAgentsQueryOptions } from "#/api/a2a";
import { PageBody, PageHeader } from "#/components/page-header";
import { Button } from "#/components/ui/button";
import { JsonCodeEditor } from "#/components/ui/json-code-editor";
import { JsonHighlight } from "#/components/ui/json-highlight";
import { Label } from "#/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "#/components/ui/tabs";
import { Textarea } from "#/components/ui/textarea";
import { GATEWAY_API_KEY, GATEWAY_API_URL } from "#/lib/gateway-base";
import { pageTitle } from "#/lib/page-meta";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/playground/a2a")({
	...pageTitle("A2A"),
	component: RouteComponent,
});

type JSONRPCResponse = {
	jsonrpc?: string;
	id?: string | number | null;
	result?: unknown;
	error?: { code?: number; message?: string; data?: unknown };
};

type A2ASkill = {
	id: string;
	name?: string;
	description?: string;
};

let rpcSeq = 1;
function nextId() {
	return rpcSeq++;
}

function pretty(value: unknown): string {
	try {
		return JSON.stringify(value, null, 2);
	} catch {
		return String(value);
	}
}

function isJsonText(text: string): boolean {
	const trimmed = text.trim();
	if (!trimmed.startsWith("{") && !trimmed.startsWith("[")) return false;
	try {
		JSON.parse(trimmed);
		return true;
	} catch {
		return false;
	}
}

function skillsFromCard(card: unknown): A2ASkill[] {
	if (!card || typeof card !== "object") return [];
	const skills = (card as { skills?: unknown }).skills;
	if (!Array.isArray(skills)) return [];
	const out: A2ASkill[] = [];
	for (const s of skills) {
		if (!s || typeof s !== "object") continue;
		const id = (s as { id?: unknown }).id;
		if (typeof id !== "string" || !id) continue;
		out.push({
			id,
			name:
				typeof (s as { name?: unknown }).name === "string"
					? (s as { name: string }).name
					: undefined,
			description:
				typeof (s as { description?: unknown }).description === "string"
					? (s as { description: string }).description
					: undefined,
		});
	}
	return out;
}

async function parseBody(res: Response): Promise<unknown> {
	const ct = (res.headers.get("Content-Type") ?? "").toLowerCase();
	const text = await res.text();
	if (!text.trim()) return null;

	if (
		ct.includes("text/event-stream") ||
		text.startsWith("event:") ||
		text.includes("\ndata:")
	) {
		const messages: unknown[] = [];
		for (const line of text.split("\n")) {
			const trimmed = line.trim();
			if (!trimmed.startsWith("data:")) continue;
			const payload = trimmed.slice(5).trim();
			if (!payload || payload === "[DONE]") continue;
			try {
				messages.push(JSON.parse(payload));
			} catch {
				messages.push(payload);
			}
		}
		if (messages.length === 1) return messages[0];
		if (messages.length > 1) return messages;
		return text;
	}

	try {
		return JSON.parse(text) as unknown;
	} catch {
		return text;
	}
}

function gatewayErrorMessage(data: unknown, status: number): string {
	if (
		typeof data === "object" &&
		data &&
		"error" in data &&
		typeof (data as { error?: { message?: string } }).error?.message ===
			"string"
	) {
		return (data as { error: { message: string } }).error.message;
	}
	return pretty(data) || `HTTP ${status}`;
}

async function a2aFetchCard(alias: string): Promise<unknown> {
	const res = await fetch(
		`${GATEWAY_API_URL}/a2a/${encodeURIComponent(alias)}/.well-known/agent-card.json`,
		{
			headers: { Authorization: `Bearer ${GATEWAY_API_KEY}` },
		},
	);
	const data = await parseBody(res);
	if (!res.ok) throw new Error(gatewayErrorMessage(data, res.status));
	return data;
}

async function a2aFetchRPC(alias: string, body: unknown): Promise<unknown> {
	const res = await fetch(
		`${GATEWAY_API_URL}/a2a/${encodeURIComponent(alias)}`,
		{
			method: "POST",
			headers: {
				Authorization: `Bearer ${GATEWAY_API_KEY}`,
				"Content-Type": "application/json",
				Accept: "application/json, text/event-stream",
			},
			body: JSON.stringify(body),
		},
	);
	const data = await parseBody(res);
	if (!res.ok) throw new Error(gatewayErrorMessage(data, res.status));
	return data;
}

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const agentsQuery = useQuery(a2aAgentsQueryOptions(orgId));
	const agents = useMemo(
		() => (agentsQuery.data ?? []).filter((a) => a.enabled),
		[agentsQuery.data],
	);

	const [alias, setAlias] = useState("");
	const [message, setMessage] = useState("Hello from AFI.");
	const [skillId, setSkillId] = useState("");
	const [skills, setSkills] = useState<A2ASkill[]>([]);
	const [rawText, setRawText] = useState(
		pretty({
			jsonrpc: "2.0",
			id: 1,
			method: "message/send",
			params: {
				message: {
					role: "user",
					parts: [{ text: "hi" }],
				},
			},
		}),
	);
	const [rawError, setRawError] = useState<string | null>(null);
	const [responseText, setResponseText] = useState("");
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const messageId = useId();
	const rawEditorId = useId();

	const splitRef = useRef<HTMLDivElement>(null);
	const [configPct, setConfigPct] = useState(58);
	const [isDragging, setIsDragging] = useState(false);

	useEffect(() => {
		if (!agents.length) {
			setAlias("");
			return;
		}
		setAlias((prev) => {
			if (agents.some((a) => a.alias === prev)) return prev;
			return agents[0].alias;
		});
	}, [agents]);

	const selectAlias = (next: string) => {
		setAlias(next);
		setSkills([]);
		setSkillId("");
		setResponseText("");
		setError(null);
	};

	useEffect(() => {
		if (!isDragging) return;
		const onMove = (e: PointerEvent) => {
			const el = splitRef.current;
			if (!el) return;
			const rect = el.getBoundingClientRect();
			if (rect.width <= 0) return;
			const next = ((e.clientX - rect.left) / rect.width) * 100;
			setConfigPct(Math.min(75, Math.max(30, next)));
		};
		const onUp = () => setIsDragging(false);
		document.body.style.cursor = "col-resize";
		document.body.style.userSelect = "none";
		window.addEventListener("pointermove", onMove);
		window.addEventListener("pointerup", onUp);
		window.addEventListener("pointercancel", onUp);
		return () => {
			document.body.style.cursor = "";
			document.body.style.userSelect = "";
			window.removeEventListener("pointermove", onMove);
			window.removeEventListener("pointerup", onUp);
			window.removeEventListener("pointercancel", onUp);
		};
	}, [isDragging]);

	const selectedSkill = skills.find((s) => s.id === skillId);

	const showResponse = (data: unknown) => {
		setResponseText(pretty(data));
		if ((data as JSONRPCResponse)?.error) {
			setError(
				(data as JSONRPCResponse).error?.message ??
					"JSON-RPC returned an error",
			);
		}
	};

	const fetchCard = async () => {
		if (!alias || busy) return;
		setBusy(true);
		setError(null);
		try {
			const data = await a2aFetchCard(alias);
			showResponse(data);
			const list = skillsFromCard(data);
			setSkills(list);
			setSkillId((prev) =>
				list.some((s) => s.id === prev) ? prev : (list[0]?.id ?? ""),
			);
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to fetch agent card");
		} finally {
			setBusy(false);
		}
	};

	const sendMessage = async () => {
		const text = message.trim();
		if (!alias || !text || busy) return;
		setBusy(true);
		setError(null);
		try {
			const params: Record<string, unknown> = {
				message: {
					role: "user",
					parts: [{ text }],
				},
			};
			if (skillId) params.skill = skillId;
			const data = await a2aFetchRPC(alias, {
				jsonrpc: "2.0",
				id: nextId(),
				method: "message/send",
				params,
			});
			showResponse(data);
		} catch (e) {
			setError(e instanceof Error ? e.message : "message/send failed");
		} finally {
			setBusy(false);
		}
	};

	const sendRaw = async () => {
		if (!alias || busy) return;
		let body: unknown;
		try {
			body = JSON.parse(rawText) as unknown;
			setRawError(null);
		} catch {
			setRawError("Request must be valid JSON");
			return;
		}
		setBusy(true);
		setError(null);
		try {
			const data = await a2aFetchRPC(alias, body);
			showResponse(data);
		} catch (e) {
			setError(e instanceof Error ? e.message : "Request failed");
		} finally {
			setBusy(false);
		}
	};

	return (
		<PageBody>
			<PageHeader
				title="A2A"
				description="Fetch agent cards and send messages through the gateway."
				info="Uses GET /a2a/{alias}/.well-known/agent-card.json and POST /a2a/{alias} with the local-dev gateway API key. Register agents under Governance → A2A."
			/>
			<div
				ref={splitRef}
				className="flex flex-col gap-6 lg:flex-row lg:items-stretch lg:gap-0"
				style={{ ["--a2a-config-pct" as string]: `${configPct}%` }}
			>
				<div className="min-w-0 space-y-5 lg:w-[var(--a2a-config-pct)] lg:shrink-0 lg:pr-3">
					{agentsQuery.isError ? (
						<p className="text-destructive text-sm">
							{agentsQuery.error instanceof Error
								? agentsQuery.error.message
								: "Failed to load A2A agents"}
						</p>
					) : null}
					{!agentsQuery.isLoading && agents.length === 0 ? (
						<p className="text-muted-foreground text-sm">
							No enabled A2A agents. Add one under{" "}
							<Link to="/app/a2a" className="underline">
								A2A
							</Link>{" "}
							(and ensure a snapshot is published).
						</p>
					) : null}

					<div className="space-y-1.5">
						<Label>Agent</Label>
						<Select value={alias} onValueChange={(v) => selectAlias(v ?? "")}>
							<SelectTrigger className="w-full">
								<SelectValue placeholder="Select agent" />
							</SelectTrigger>
							<SelectContent empty="No agents">
								{agents.map((a) => (
									<SelectItem key={a.id} value={a.alias}>
										{a.name} ({a.alias})
									</SelectItem>
								))}
							</SelectContent>
						</Select>
						{alias ? (
							<p className="text-muted-foreground text-xs">
								<code className="text-xs">
									POST {GATEWAY_API_URL}/a2a/{alias}
								</code>
							</p>
						) : null}
					</div>

					<Tabs defaultValue="message">
						<TabsList>
							<TabsTrigger value="message">Message</TabsTrigger>
							<TabsTrigger value="raw">Raw JSON-RPC</TabsTrigger>
						</TabsList>

						<TabsContent value="message" className="space-y-5 pt-4">
							<div className="flex flex-wrap gap-2">
								<Button
									variant="outline"
									onClick={() => void fetchCard()}
									disabled={busy || !alias}
								>
									{busy ? <Loader2Icon className="animate-spin" /> : null}
									Fetch agent card
								</Button>
							</div>

							{skills.length > 0 ? (
								<div className="space-y-1.5">
									<Label>Skill (optional)</Label>
									<Select
										value={skillId}
										onValueChange={(v) => setSkillId(v ?? "")}
									>
										<SelectTrigger className="w-full">
											<SelectValue placeholder="No skill" />
										</SelectTrigger>
										<SelectContent>
											{skills.map((s) => (
												<SelectItem key={s.id} value={s.id}>
													{s.name ? `${s.name} (${s.id})` : s.id}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
									{selectedSkill?.description ? (
										<p className="text-muted-foreground text-xs">
											{selectedSkill.description}
										</p>
									) : null}
								</div>
							) : null}

							<div className="space-y-1.5">
								<Label htmlFor={messageId}>Message</Label>
								<Textarea
									id={messageId}
									value={message}
									onChange={(e) => setMessage(e.target.value)}
									rows={8}
									className="min-h-40 text-base"
									placeholder="Say something…"
								/>
							</div>

							{error ? (
								<pre className="text-destructive max-h-40 overflow-auto rounded-md border border-destructive/30 bg-destructive/5 p-3 text-xs whitespace-pre-wrap">
									{error}
								</pre>
							) : null}

							<Button
								size="lg"
								onClick={() => void sendMessage()}
								disabled={busy || !alias || !message.trim()}
							>
								{busy ? (
									<>
										<Loader2Icon className="animate-spin" />
										Sending…
									</>
								) : (
									"Send message"
								)}
							</Button>
						</TabsContent>

						<TabsContent value="raw" className="space-y-5 pt-4">
							<div className="space-y-1.5">
								<div className="flex items-center justify-between gap-2">
									<Label htmlFor={rawEditorId}>Request</Label>
									<span className="text-[11px] text-muted-foreground">
										JSON-RPC body
									</span>
								</div>
								<JsonCodeEditor
									id={rawEditorId}
									value={rawText}
									onChange={(v) => {
										setRawText(v);
										setRawError(null);
									}}
									minHeight="16rem"
									invalid={Boolean(rawError)}
								/>
								{rawError ? (
									<p className="text-destructive text-xs">{rawError}</p>
								) : null}
							</div>

							{error ? (
								<pre className="text-destructive max-h-40 overflow-auto rounded-md border border-destructive/30 bg-destructive/5 p-3 text-xs whitespace-pre-wrap">
									{error}
								</pre>
							) : null}

							<Button
								size="lg"
								onClick={() => void sendRaw()}
								disabled={busy || !alias}
							>
								{busy ? (
									<>
										<Loader2Icon className="animate-spin" />
										Sending…
									</>
								) : (
									"Send"
								)}
							</Button>
						</TabsContent>
					</Tabs>
				</div>

				<button
					type="button"
					aria-label="Resize panels"
					className="hidden lg:flex w-3 shrink-0 cursor-col-resize items-stretch justify-center self-stretch px-0"
					onPointerDown={(e) => {
						e.preventDefault();
						setIsDragging(true);
					}}
				>
					<span
						className={`bg-border my-1 w-px rounded-full transition-colors ${
							isDragging ? "bg-foreground/40" : "hover:bg-foreground/30"
						}`}
					/>
				</button>

				<div className="bg-muted/30 min-w-0 flex-1 space-y-3 rounded-xl border p-5 lg:pl-3">
					<h3 className="text-sm font-medium">Response</h3>
					<p className="text-muted-foreground text-sm">
						Agent card or JSON-RPC result appears here.
					</p>
					{responseText ? (
						isJsonText(responseText) ? (
							<JsonHighlight value={responseText} />
						) : (
							<pre className="bg-background max-h-[32rem] overflow-auto rounded-lg border p-3 text-xs whitespace-pre-wrap">
								{responseText}
							</pre>
						)
					) : (
						<div className="text-muted-foreground flex min-h-32 items-center justify-center rounded-lg border border-dashed text-sm">
							No response yet
						</div>
					)}
				</div>
			</div>
		</PageBody>
	);
}
