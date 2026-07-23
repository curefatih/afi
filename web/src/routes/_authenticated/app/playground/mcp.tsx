import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Loader2Icon } from "lucide-react";
import { useEffect, useId, useMemo, useRef, useState } from "react";
import { mcpBackendsQueryOptions } from "#/api/mcp";
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
import { GATEWAY_API_KEY, GATEWAY_API_URL } from "#/lib/gateway-base";
import { pageTitle } from "#/lib/page-meta";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/playground/mcp")({
	...pageTitle("MCP"),
	component: RouteComponent,
});

type MCPTool = {
	name: string;
	description?: string;
	inputSchema?: Record<string, unknown>;
};

type JSONRPCResponse = {
	jsonrpc?: string;
	id?: string | number | null;
	result?: unknown;
	error?: { code?: number; message?: string; data?: unknown };
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

function defaultArgsFromSchema(schema?: Record<string, unknown>): string {
	if (!schema || typeof schema !== "object") return "{\n  \n}";
	const props = schema.properties;
	if (!props || typeof props !== "object" || Array.isArray(props)) {
		return "{\n  \n}";
	}
	const required = Array.isArray(schema.required)
		? new Set(schema.required.filter((r): r is string => typeof r === "string"))
		: null;
	const out: Record<string, unknown> = {};
	for (const [key, def] of Object.entries(props as Record<string, unknown>)) {
		if (required && !required.has(key)) continue;
		const type =
			def && typeof def === "object" && "type" in def
				? (def as { type?: unknown }).type
				: undefined;
		if (type === "string") out[key] = "";
		else if (type === "number" || type === "integer") out[key] = 0;
		else if (type === "boolean") out[key] = false;
		else if (type === "array") out[key] = [];
		else if (type === "object") out[key] = {};
		else out[key] = null;
	}
	return pretty(out);
}

async function parseMCPBody(res: Response): Promise<unknown> {
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

async function mcpFetch(
	alias: string,
	body: unknown,
	sessionId: string | null,
): Promise<{ status: number; sessionId: string | null; data: unknown }> {
	const headers: Record<string, string> = {
		Authorization: `Bearer ${GATEWAY_API_KEY}`,
		"Content-Type": "application/json",
		Accept: "application/json, text/event-stream",
	};
	if (sessionId) headers["Mcp-Session-Id"] = sessionId;

	const res = await fetch(
		`${GATEWAY_API_URL}/mcp/${encodeURIComponent(alias)}`,
		{
			method: "POST",
			headers,
			body: JSON.stringify(body),
		},
	);

	const nextSession = res.headers.get("Mcp-Session-Id") ?? sessionId;
	const data = await parseMCPBody(res);
	if (!res.ok) {
		const msg =
			typeof data === "object" &&
			data &&
			"error" in data &&
			typeof (data as { error?: { message?: string } }).error?.message ===
				"string"
				? (data as { error: { message: string } }).error.message
				: pretty(data) || `HTTP ${res.status}`;
		throw new Error(msg);
	}
	return { status: res.status, sessionId: nextSession, data };
}

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const backendsQuery = useQuery(mcpBackendsQueryOptions(orgId));
	const backends = useMemo(
		() => (backendsQuery.data ?? []).filter((b) => b.enabled),
		[backendsQuery.data],
	);

	const [alias, setAlias] = useState("");
	const [sessionId, setSessionId] = useState<string | null>(null);
	const [tools, setTools] = useState<MCPTool[]>([]);
	const [toolName, setToolName] = useState("");
	const [argsText, setArgsText] = useState("{\n  \n}");
	const [argsError, setArgsError] = useState<string | null>(null);
	const [rawText, setRawText] = useState(
		pretty({
			jsonrpc: "2.0",
			id: 1,
			method: "tools/list",
			params: {},
		}),
	);
	const [rawError, setRawError] = useState<string | null>(null);
	const [responseText, setResponseText] = useState("");
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const sessionByAlias = useRef<Map<string, string>>(new Map());
	const argsEditorId = useId();
	const rawEditorId = useId();

	useEffect(() => {
		if (!backends.length) {
			setAlias("");
			return;
		}
		setAlias((prev) =>
			backends.some((b) => b.alias === prev) ? prev : backends[0].alias,
		);
	}, [backends]);

	useEffect(() => {
		setTools([]);
		setToolName("");
		setArgsText("{\n  \n}");
		setArgsError(null);
		setResponseText("");
		setError(null);
		setSessionId(alias ? (sessionByAlias.current.get(alias) ?? null) : null);
	}, [alias]);

	const selectedTool = tools.find((t) => t.name === toolName);

	const ensureSession = async (
		currentAlias: string,
		currentSession: string | null,
	) => {
		if (currentSession) return currentSession;
		const init = await mcpFetch(
			currentAlias,
			{
				jsonrpc: "2.0",
				id: nextId(),
				method: "initialize",
				params: {
					protocolVersion: "2025-03-26",
					capabilities: {},
					clientInfo: { name: "afi-playground", version: "0.1.0" },
				},
			},
			null,
		);
		if (init.sessionId) {
			sessionByAlias.current.set(currentAlias, init.sessionId);
			setSessionId(init.sessionId);
		}
		try {
			await mcpFetch(
				currentAlias,
				{ jsonrpc: "2.0", method: "notifications/initialized" },
				init.sessionId,
			);
		} catch {
			// Some servers don't require or accept the notification.
		}
		return init.sessionId;
	};

	const listTools = async () => {
		if (!alias || busy) return;
		setBusy(true);
		setError(null);
		try {
			const sid = await ensureSession(alias, sessionId);
			const { sessionId: nextSid, data } = await mcpFetch(
				alias,
				{
					jsonrpc: "2.0",
					id: nextId(),
					method: "tools/list",
					params: {},
				},
				sid,
			);
			if (nextSid) {
				sessionByAlias.current.set(alias, nextSid);
				setSessionId(nextSid);
			}
			setResponseText(pretty(data));
			const result = (data as JSONRPCResponse)?.result as
				| { tools?: MCPTool[] }
				| undefined;
			const list = Array.isArray(result?.tools) ? result.tools : [];
			setTools(list);
			const nextTool =
				list.find((t) => t.name === toolName)?.name ?? list[0]?.name ?? "";
			setToolName(nextTool);
			const tool = list.find((t) => t.name === nextTool);
			if (tool) setArgsText(defaultArgsFromSchema(tool.inputSchema));
			if ((data as JSONRPCResponse)?.error) {
				setError(
					(data as JSONRPCResponse).error?.message ??
						"tools/list returned an error",
				);
			}
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to list tools");
		} finally {
			setBusy(false);
		}
	};

	const callTool = async () => {
		if (!alias || !toolName || busy) return;
		let args: unknown;
		try {
			args = JSON.parse(argsText) as unknown;
			setArgsError(null);
		} catch {
			setArgsError("Arguments must be valid JSON");
			return;
		}
		setBusy(true);
		setError(null);
		try {
			const sid = await ensureSession(alias, sessionId);
			const { sessionId: nextSid, data } = await mcpFetch(
				alias,
				{
					jsonrpc: "2.0",
					id: nextId(),
					method: "tools/call",
					params: { name: toolName, arguments: args },
				},
				sid,
			);
			if (nextSid) {
				sessionByAlias.current.set(alias, nextSid);
				setSessionId(nextSid);
			}
			setResponseText(pretty(data));
			if ((data as JSONRPCResponse)?.error) {
				setError(
					(data as JSONRPCResponse).error?.message ??
						"tools/call returned an error",
				);
			}
		} catch (e) {
			setError(e instanceof Error ? e.message : "Tool call failed");
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
			const { sessionId: nextSid, data } = await mcpFetch(
				alias,
				body,
				sessionId,
			);
			if (nextSid) {
				sessionByAlias.current.set(alias, nextSid);
				setSessionId(nextSid);
			}
			setResponseText(pretty(data));
			if ((data as JSONRPCResponse)?.error) {
				setError(
					(data as JSONRPCResponse).error?.message ??
						"JSON-RPC returned an error",
				);
			}
		} catch (e) {
			setError(e instanceof Error ? e.message : "Request failed");
		} finally {
			setBusy(false);
		}
	};

	const onToolChange = (name: string) => {
		setToolName(name);
		const tool = tools.find((t) => t.name === name);
		if (tool) setArgsText(defaultArgsFromSchema(tool.inputSchema));
	};

	const splitRef = useRef<HTMLDivElement>(null);
	const [configPct, setConfigPct] = useState(58);
	const [isDragging, setIsDragging] = useState(false);

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

	return (
		<PageBody>
			<PageHeader
				title="MCP"
				description="List and call tools on gateway MCP backends."
				info="Uses POST /mcp/{alias} with the local-dev gateway API key. Register backends under Governance → MCP."
			/>
			<div
				ref={splitRef}
				className="flex flex-col gap-6 lg:flex-row lg:items-stretch lg:gap-0"
				style={{ ["--mcp-config-pct" as string]: `${configPct}%` }}
			>
				<div className="min-w-0 space-y-5 lg:w-[var(--mcp-config-pct)] lg:shrink-0 lg:pr-3">
					{backendsQuery.isError ? (
						<p className="text-destructive text-sm">
							{backendsQuery.error instanceof Error
								? backendsQuery.error.message
								: "Failed to load MCP backends"}
						</p>
					) : null}
					{!backendsQuery.isLoading && backends.length === 0 ? (
						<p className="text-muted-foreground text-sm">
							No enabled MCP backends. Add one under{" "}
							<Link to="/app/mcp" className="underline">
								MCP
							</Link>{" "}
							(and ensure a snapshot is published).
						</p>
					) : null}

					<div className="space-y-1.5">
						<Label>Backend</Label>
						<Select value={alias} onValueChange={(v) => setAlias(v ?? "")}>
							<SelectTrigger className="w-full">
								<SelectValue placeholder="Select backend" />
							</SelectTrigger>
							<SelectContent empty="No backends">
								{backends.map((b) => (
									<SelectItem key={b.id} value={b.alias}>
										{b.name} ({b.alias})
									</SelectItem>
								))}
							</SelectContent>
						</Select>
						{alias ? (
							<p className="text-muted-foreground text-xs">
								<code className="text-xs">
									POST {GATEWAY_API_URL}/mcp/{alias}
								</code>
								{sessionId ? (
									<>
										{" · "}
										session <code className="text-xs">{sessionId}</code>
									</>
								) : null}
							</p>
						) : null}
					</div>

					<Tabs defaultValue="tools">
						<TabsList>
							<TabsTrigger value="tools">Tools</TabsTrigger>
							<TabsTrigger value="raw">Raw JSON-RPC</TabsTrigger>
						</TabsList>

						<TabsContent value="tools" className="space-y-5 pt-4">
							<div className="flex flex-wrap gap-2">
								<Button
									variant="outline"
									onClick={() => void listTools()}
									disabled={busy || !alias}
								>
									{busy ? <Loader2Icon className="animate-spin" /> : null}
									List tools
								</Button>
							</div>

							<div className="space-y-1.5">
								<Label>Tool</Label>
								<Select
									value={toolName}
									onValueChange={(v) => onToolChange(v ?? "")}
									disabled={tools.length === 0}
								>
									<SelectTrigger className="w-full">
										<SelectValue placeholder="List tools first" />
									</SelectTrigger>
									<SelectContent empty="No tools loaded">
										{tools.map((t) => (
											<SelectItem key={t.name} value={t.name}>
												{t.name}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
								{selectedTool?.description ? (
									<p className="text-muted-foreground text-xs">
										{selectedTool.description}
									</p>
								) : null}
							</div>

							<div className="space-y-1.5">
								<div className="flex items-center justify-between gap-2">
									<Label htmlFor={argsEditorId}>Arguments</Label>
									<span className="text-[11px] text-muted-foreground">
										Tab indent · Shift+Tab outdent
									</span>
								</div>
								<JsonCodeEditor
									id={argsEditorId}
									value={argsText}
									onChange={(v) => {
										setArgsText(v);
										setArgsError(null);
									}}
									minHeight="12rem"
									invalid={Boolean(argsError)}
									placeholder="{ }"
								/>
								{argsError ? (
									<p className="text-destructive text-xs">{argsError}</p>
								) : null}
							</div>

							{error ? (
								<pre className="text-destructive max-h-40 overflow-auto rounded-md border border-destructive/30 bg-destructive/5 p-3 text-xs whitespace-pre-wrap">
									{error}
								</pre>
							) : null}

							<Button
								size="lg"
								onClick={() => void callTool()}
								disabled={busy || !alias || !toolName}
							>
								{busy ? (
									<>
										<Loader2Icon className="animate-spin" />
										Calling…
									</>
								) : (
									"Call tool"
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
						Gateway JSON-RPC result (or SSE payloads) appears here.
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
