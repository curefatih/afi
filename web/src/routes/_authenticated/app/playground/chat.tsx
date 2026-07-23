import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	ArrowUpIcon,
	Loader2Icon,
	MessageCircleDashedIcon,
	RotateCwIcon,
	Settings2Icon,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { ApiKey } from "#/api/keys";
import { PageBody } from "#/components/page-header";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardAction,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";
import {
	Empty,
	EmptyDescription,
	EmptyHeader,
	EmptyMedia,
	EmptyTitle,
} from "#/components/ui/empty";
import { Input } from "#/components/ui/input";
import {
	InputGroup,
	InputGroupAddon,
	InputGroupButton,
	InputGroupTextarea,
} from "#/components/ui/input-group";
import { JsonCodeEditor } from "#/components/ui/json-code-editor";
import { Label } from "#/components/ui/label";
import { MessageAnimated } from "#/components/ui/message-animated";
import {
	MessageScroller,
	MessageScrollerButton,
	MessageScrollerContent,
	MessageScrollerProvider,
	MessageScrollerViewport,
} from "#/components/ui/message-scroller";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { Separator } from "#/components/ui/separator";
import {
	Sheet,
	SheetContent,
	SheetDescription,
	SheetHeader,
	SheetTitle,
	SheetTrigger,
} from "#/components/ui/sheet";
import { Slider } from "#/components/ui/slider";
import { Switch } from "#/components/ui/switch";
import { Textarea } from "#/components/ui/textarea";
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "#/components/ui/tooltip";
import { ApiError, apiFetch } from "#/lib/api-client";
import {
	GATEWAY_API_URL,
	isSeededPlaygroundProject,
	PLAYGROUND_SEEDED_KEY,
	resolvePlaygroundApiKey,
	storePlaygroundApiKey,
} from "#/lib/gateway-base";
import { type GatewayModel, isChatModel } from "#/lib/gateway-models";
import { pageTitle } from "#/lib/page-meta";
import {
	useActiveOrg,
	useActiveProject,
	useOrgActions,
} from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/playground/chat")({
	...pageTitle("Chat"),
	component: RouteComponent,
});

type Message = {
	id: string;
	role: "user" | "assistant";
	parts: Array<{
		type: "text";
		text: string;
	}>;
};

type ResponseFormat = "text" | "json_object" | "json_schema";

const DEFAULT_JSON_SCHEMA = `{
  "type": "object",
  "properties": {
    "answer": { "type": "string" }
  },
  "required": ["answer"],
  "additionalProperties": false
}`;

const PLAYGROUND_KEY_NAME = "Playground";

async function readSSEContent(
	res: Response,
	onDelta: (text: string) => void,
): Promise<void> {
	const reader = res.body?.getReader();
	if (!reader) {
		throw new Error("No response body");
	}
	const decoder = new TextDecoder();
	let buffer = "";
	while (true) {
		const { done, value } = await reader.read();
		if (done) break;
		buffer += decoder.decode(value, { stream: true });
		const parts = buffer.split("\n");
		buffer = parts.pop() ?? "";
		for (const line of parts) {
			const trimmed = line.trim();
			if (!trimmed.startsWith("data:")) continue;
			const payload = trimmed.slice(5).trim();
			if (!payload || payload === "[DONE]") continue;
			try {
				const chunk = JSON.parse(payload) as {
					choices?: Array<{ delta?: { content?: string } }>;
				};
				const piece = chunk.choices?.[0]?.delta?.content;
				if (piece) onDelta(piece);
			} catch {
				// skip malformed chunk lines
			}
		}
	}
}

function parseOptionalInt(raw: string): number | undefined {
	const trimmed = raw.trim();
	if (!trimmed) return undefined;
	const n = Number(trimmed);
	if (!Number.isFinite(n) || !Number.isInteger(n) || n < 1) return undefined;
	return n;
}

function sliderValue(
	value: number | readonly number[],
	fallback: number,
): number {
	if (typeof value === "number") return value;
	return value[0] ?? fallback;
}

function projectKeyErrorMessage(error: unknown): string {
	if (error instanceof ApiError && error.status === 403) {
		return "Org admin required to provision a project key for the playground";
	}
	if (error instanceof Error && error.message) return error.message;
	return "Failed to provision playground key";
}

function SettingsFields({
	model,
	models,
	modelsError,
	projectId,
	projects,
	keyStatus,
	keyError,
	systemPrompt,
	temperature,
	topP,
	maxTokens,
	responseFormat,
	jsonSchemaName,
	jsonSchemaStrict,
	jsonSchemaText,
	jsonSchemaError,
	onModelChange,
	onProjectChange,
	onSystemPromptChange,
	onTemperatureChange,
	onTopPChange,
	onMaxTokensChange,
	onResponseFormatChange,
	onJsonSchemaNameChange,
	onJsonSchemaStrictChange,
	onJsonSchemaTextChange,
}: {
	model: string;
	models: GatewayModel[];
	modelsError: string | null;
	projectId: string;
	projects: Array<{ id: string; name: string }>;
	keyStatus: "ready" | "provisioning" | "error";
	keyError: string | null;
	systemPrompt: string;
	temperature: number;
	topP: number;
	maxTokens: string;
	responseFormat: ResponseFormat;
	jsonSchemaName: string;
	jsonSchemaStrict: boolean;
	jsonSchemaText: string;
	jsonSchemaError: string | null;
	onModelChange: (value: string) => void;
	onProjectChange: (value: string) => void;
	onSystemPromptChange: (value: string) => void;
	onTemperatureChange: (value: number) => void;
	onTopPChange: (value: number) => void;
	onMaxTokensChange: (value: string) => void;
	onResponseFormatChange: (value: ResponseFormat) => void;
	onJsonSchemaNameChange: (value: string) => void;
	onJsonSchemaStrictChange: (value: boolean) => void;
	onJsonSchemaTextChange: (value: string) => void;
}) {
	const usingSeededKey = isSeededPlaygroundProject(projectId);

	return (
		<div className="space-y-4">
			<div className="space-y-2">
				<Label htmlFor="chat-model">Model</Label>
				{modelsError ? (
					<p className="text-destructive text-xs">{modelsError}</p>
				) : null}
				<Select
					value={model}
					onValueChange={(value) => onModelChange(value ?? models[0]?.id ?? "")}
					disabled={models.length === 0}
				>
					<SelectTrigger id="chat-model" className="w-full">
						<SelectValue placeholder="Loading routes…" />
					</SelectTrigger>
					<SelectContent>
						{models.map((m) => (
							<SelectItem key={m.id} value={m.id}>
								{m.id}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
				<p className="text-muted-foreground text-xs">
					From gateway /v1/models (mode=chat). Streaming follows each
					model&apos;s supports_streaming capability.
				</p>
			</div>

			<div className="space-y-2">
				<Label htmlFor="chat-project">Project context</Label>
				<Select
					value={projectId}
					onValueChange={(value) =>
						onProjectChange(value ?? PLAYGROUND_SEEDED_KEY)
					}
				>
					<SelectTrigger id="chat-project" className="w-full">
						<SelectValue placeholder="Seeded key" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value={PLAYGROUND_SEEDED_KEY}>
							Seeded local key
						</SelectItem>
						{projects.map((project) => (
							<SelectItem key={project.id} value={project.id}>
								{project.name}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
				{keyStatus === "provisioning" ? (
					<p className="flex items-center gap-1.5 text-muted-foreground text-xs">
						<Loader2Icon className="size-3 animate-spin" />
						Provisioning project key…
					</p>
				) : keyError ? (
					<p className="text-destructive text-xs">{keyError}</p>
				) : (
					<p className="text-muted-foreground text-xs">
						{usingSeededKey
							? "Uses the seeded local-dev key (Local Project)."
							: "Auto-provisions a project service-account key for this session (org admin). Usage follows that project."}
					</p>
				)}
			</div>

			<Separator />

			<div className="space-y-2">
				<Label htmlFor="chat-system">System prompt</Label>
				<Textarea
					id="chat-system"
					value={systemPrompt}
					onChange={(e) => onSystemPromptChange(e.target.value)}
					placeholder="Optional instructions for the model…"
					rows={3}
					className="min-h-20 resize-y font-mono text-xs"
				/>
			</div>

			<div className="space-y-2">
				<div className="flex items-center justify-between gap-2">
					<Label htmlFor="chat-temperature">Temperature</Label>
					<span className="tabular-nums text-muted-foreground text-xs">
						{temperature.toFixed(2)}
					</span>
				</div>
				<Slider
					id="chat-temperature"
					min={0}
					max={2}
					step={0.01}
					value={[temperature]}
					onValueChange={(value) => onTemperatureChange(sliderValue(value, 1))}
				/>
				<p className="text-muted-foreground text-xs">
					0 = deterministic · 2 = more random
				</p>
			</div>

			<div className="space-y-2">
				<div className="flex items-center justify-between gap-2">
					<Label htmlFor="chat-top-p">Top P</Label>
					<span className="tabular-nums text-muted-foreground text-xs">
						{topP.toFixed(2)}
					</span>
				</div>
				<Slider
					id="chat-top-p"
					min={0}
					max={1}
					step={0.01}
					value={[topP]}
					onValueChange={(value) => onTopPChange(sliderValue(value, 1))}
				/>
				<p className="text-muted-foreground text-xs">
					Nucleus sampling. 1 keeps the full distribution.
				</p>
			</div>

			<div className="space-y-2">
				<Label htmlFor="chat-max-tokens">Max tokens</Label>
				<Input
					id="chat-max-tokens"
					type="number"
					min={1}
					step={1}
					inputMode="numeric"
					placeholder="Provider default"
					value={maxTokens}
					onChange={(e) => onMaxTokensChange(e.target.value)}
				/>
				<p className="text-muted-foreground text-xs">
					Leave empty to omit max_tokens from the request.
				</p>
			</div>

			<Separator />

			<div className="space-y-2">
				<Label htmlFor="chat-response-format">Response format</Label>
				<Select
					value={responseFormat}
					onValueChange={(value) =>
						onResponseFormatChange((value as ResponseFormat) ?? "text")
					}
				>
					<SelectTrigger id="chat-response-format" className="w-full">
						<SelectValue />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="text">Text</SelectItem>
						<SelectItem value="json_object">JSON object</SelectItem>
						<SelectItem value="json_schema">JSON schema</SelectItem>
					</SelectContent>
				</Select>
				<p className="text-muted-foreground text-xs">
					Structured outputs use OpenAI response_format. Support varies by
					upstream provider.
				</p>
			</div>

			{responseFormat === "json_schema" ? (
				<div className="space-y-4">
					<div className="space-y-2">
						<Label htmlFor="chat-schema-name">Schema name</Label>
						<Input
							id="chat-schema-name"
							value={jsonSchemaName}
							onChange={(e) => onJsonSchemaNameChange(e.target.value)}
							placeholder="response"
						/>
					</div>

					<div className="flex items-center justify-between gap-3">
						<div className="space-y-0.5">
							<Label htmlFor="chat-schema-strict">Strict</Label>
							<p className="text-muted-foreground text-xs">
								Require adherence to the schema
							</p>
						</div>
						<Switch
							id="chat-schema-strict"
							checked={jsonSchemaStrict}
							onCheckedChange={onJsonSchemaStrictChange}
						/>
					</div>

					<div className="space-y-2">
						<div className="flex items-center justify-between gap-2">
							<Label htmlFor="chat-json-schema">JSON schema</Label>
							<span className="text-[11px] text-muted-foreground">
								Tab indent · Shift+Tab outdent
							</span>
						</div>
						<JsonCodeEditor
							id="chat-json-schema"
							value={jsonSchemaText}
							onChange={onJsonSchemaTextChange}
							minHeight="14rem"
							invalid={Boolean(jsonSchemaError)}
							placeholder='{ "type": "object", ... }'
						/>
						{jsonSchemaError ? (
							<p className="text-destructive text-xs">{jsonSchemaError}</p>
						) : (
							<p className="text-muted-foreground text-xs">
								Sent as response_format.json_schema.schema
							</p>
						)}
					</div>
				</div>
			) : null}

			{responseFormat === "json_object" ? (
				<p className="text-muted-foreground text-xs">
					Some providers require the word &quot;json&quot; in the prompt when
					using JSON object mode.
				</p>
			) : null}
		</div>
	);
}

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const activeProject = useActiveProject();
	const { setActiveProjectById } = useOrgActions();
	const queryClient = useQueryClient();
	const [input, setInput] = useState("");
	const [models, setModels] = useState<GatewayModel[]>([]);
	const [model, setModel] = useState("");
	const [modelsError, setModelsError] = useState<string | null>(null);
	const [isBusy, setIsBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [messages, setMessages] = useState<Message[]>([]);
	const [projectId, setProjectId] = useState<string>(() => {
		if (activeProject?.id) return activeProject.id;
		return PLAYGROUND_SEEDED_KEY;
	});
	const [apiKey, setApiKey] = useState(() =>
		resolvePlaygroundApiKey(
			activeProject?.id ? activeProject.id : PLAYGROUND_SEEDED_KEY,
		),
	);
	const [keyStatus, setKeyStatus] = useState<
		"ready" | "provisioning" | "error"
	>(() =>
		resolvePlaygroundApiKey(
			activeProject?.id ? activeProject.id : PLAYGROUND_SEEDED_KEY,
		)
			? "ready"
			: "provisioning",
	);
	const [keyError, setKeyError] = useState<string | null>(null);
	const [systemPrompt, setSystemPrompt] = useState("");
	const [temperature, setTemperature] = useState(1);
	const [topP, setTopP] = useState(1);
	const [maxTokens, setMaxTokens] = useState("");
	const [responseFormat, setResponseFormat] = useState<ResponseFormat>("text");
	const [jsonSchemaName, setJsonSchemaName] = useState("response");
	const [jsonSchemaStrict, setJsonSchemaStrict] = useState(true);
	const [jsonSchemaText, setJsonSchemaText] = useState(DEFAULT_JSON_SCHEMA);
	const [jsonSchemaError, setJsonSchemaError] = useState<string | null>(null);
	const textareaRef = useRef<HTMLTextAreaElement>(null);
	const provisionGen = useRef(0);

	const projects = activeOrg?.projects ?? [];
	const orgId = activeOrg?.id ?? "";
	const resolvedApiKey = apiKey.trim();

	const waitingForFirstToken =
		isBusy &&
		messages.length > 0 &&
		messages[messages.length - 1]?.role === "assistant" &&
		!(messages[messages.length - 1]?.parts[0]?.text ?? "");

	useEffect(() => {
		const gen = ++provisionGen.current;
		let cancelled = false;

		const applyKey = (nextKey: string) => {
			if (cancelled || gen !== provisionGen.current) return;
			setApiKey(nextKey);
			setKeyStatus("ready");
			setKeyError(null);
		};

		const failKey = (message: string) => {
			if (cancelled || gen !== provisionGen.current) return;
			setApiKey("");
			setKeyStatus("error");
			setKeyError(message);
		};

		if (isSeededPlaygroundProject(projectId)) {
			applyKey(resolvePlaygroundApiKey(projectId));
			return () => {
				cancelled = true;
			};
		}

		const cached = resolvePlaygroundApiKey(projectId);
		if (cached) {
			applyKey(cached);
			return () => {
				cancelled = true;
			};
		}

		setKeyStatus("provisioning");
		setKeyError(null);
		setApiKey("");

		void (async () => {
			try {
				const created = await apiFetch<ApiKey>(
					`/api/v1/platform/projects/${projectId}/keys`,
					{
						method: "POST",
						body: { name: PLAYGROUND_KEY_NAME },
					},
				);
				const secret = created.key?.trim();
				if (!secret) {
					failKey("Playground key created but secret was not returned");
					return;
				}
				storePlaygroundApiKey(projectId, secret);
				if (orgId) {
					void queryClient.invalidateQueries({
						queryKey: ["organizations", orgId, "keys"],
					});
				}
				void queryClient.invalidateQueries({
					queryKey: ["projects", projectId, "keys"],
				});
				applyKey(secret);
			} catch (e: unknown) {
				failKey(projectKeyErrorMessage(e));
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [projectId, queryClient, orgId]);

	useEffect(() => {
		let cancelled = false;
		if (keyStatus === "provisioning") {
			setModels([]);
			setModel("");
			setModelsError(null);
			return;
		}
		if (!resolvedApiKey) {
			setModels([]);
			setModel("");
			setModelsError(keyError ?? "Project key unavailable");
			return;
		}
		(async () => {
			try {
				const res = await fetch(`${GATEWAY_API_URL}/v1/models`, {
					headers: { Authorization: `Bearer ${resolvedApiKey}` },
				});
				if (!res.ok) {
					throw new Error(`models HTTP ${res.status}`);
				}
				const data = (await res.json()) as {
					data?: GatewayModel[];
				};
				if (cancelled) return;
				const list = (data.data ?? []).filter(isChatModel).map((m) => ({
					id: m.id,
					mode: m.mode ?? "chat",
					supports_streaming: m.supports_streaming !== false,
					capabilities: m.capabilities,
				}));
				setModels(list);
				setModel((prev) =>
					list.some((m) => m.id === prev) ? prev : list[0]?.id || "",
				);
				setModelsError(null);
			} catch (e) {
				if (!cancelled) {
					setModelsError(
						e instanceof Error ? e.message : "Failed to load models",
					);
				}
			}
		})();
		return () => {
			cancelled = true;
		};
	}, [resolvedApiKey, keyStatus, keyError]);

	useEffect(() => {
		if (responseFormat !== "json_schema") {
			setJsonSchemaError(null);
			return;
		}
		try {
			const parsed: unknown = JSON.parse(jsonSchemaText);
			if (
				parsed === null ||
				typeof parsed !== "object" ||
				Array.isArray(parsed)
			) {
				setJsonSchemaError("Schema must be a JSON object");
				return;
			}
			setJsonSchemaError(null);
		} catch {
			setJsonSchemaError("Invalid JSON");
		}
	}, [responseFormat, jsonSchemaText]);

	const selectedModel = models.find((m) => m.id === model);
	const useStream = selectedModel?.supports_streaming !== false;

	const send = async () => {
		const text = input.trim();
		if (!text || isBusy || !model) return;

		if (!resolvedApiKey || keyStatus !== "ready") {
			setError(keyError ?? "Project key unavailable");
			return;
		}

		if (responseFormat === "json_schema") {
			if (jsonSchemaError) {
				setError(jsonSchemaError);
				return;
			}
			try {
				JSON.parse(jsonSchemaText);
			} catch {
				setError("JSON schema is invalid");
				return;
			}
		}

		const userMsg: Message = {
			id: crypto.randomUUID(),
			role: "user",
			parts: [{ type: "text", text }],
		};
		const assistantId = crypto.randomUUID();
		const next = [...messages, userMsg];
		setMessages([
			...next,
			{
				id: assistantId,
				role: "assistant",
				parts: [{ type: "text", text: "" }],
			},
		]);
		setInput("");
		setIsBusy(true);
		setError(null);

		const appendAssistant = (piece: string) => {
			setMessages((prev) =>
				prev.map((m) =>
					m.id === assistantId
						? {
								...m,
								parts: [
									{
										type: "text",
										text: (m.parts[0]?.text ?? "") + piece,
									},
								],
							}
						: m,
				),
			);
		};

		const requestMessages: Array<{ role: string; content: string }> = [];
		const system = systemPrompt.trim();
		if (system) {
			requestMessages.push({ role: "system", content: system });
		}
		for (const m of next) {
			requestMessages.push({
				role: m.role,
				content: m.parts.map((c) => c.text).join("\n"),
			});
		}

		const body: Record<string, unknown> = {
			model,
			stream: useStream,
			messages: requestMessages,
			temperature,
			top_p: topP,
		};

		const parsedMaxTokens = parseOptionalInt(maxTokens);
		if (parsedMaxTokens !== undefined) {
			body.max_tokens = parsedMaxTokens;
		}

		if (responseFormat === "json_object") {
			body.response_format = { type: "json_object" };
		} else if (responseFormat === "json_schema") {
			body.response_format = {
				type: "json_schema",
				json_schema: {
					name: jsonSchemaName.trim() || "response",
					strict: jsonSchemaStrict,
					schema: JSON.parse(jsonSchemaText) as Record<string, unknown>,
				},
			};
		}

		try {
			const res = await fetch(`${GATEWAY_API_URL}/v1/chat/completions`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${resolvedApiKey}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify(body),
			});
			if (!res.ok) {
				const errBody = await res.text();
				throw new Error(errBody || `HTTP ${res.status}`);
			}

			if (useStream) {
				await readSSEContent(res, appendAssistant);
			} else {
				const data = (await res.json()) as {
					choices?: Array<{ message?: { content?: string } }>;
				};
				const content =
					data?.choices?.[0]?.message?.content ?? "(empty response)";
				appendAssistant(String(content));
			}
		} catch (e) {
			setError(e instanceof Error ? e.message : "Request failed");
			setMessages((prev) =>
				prev.filter(
					(m) => m.id !== assistantId || (m.parts[0]?.text ?? "").length > 0,
				),
			);
		} finally {
			setIsBusy(false);
			textareaRef.current?.focus();
		}
	};

	const onProjectChange = (next: string) => {
		setProjectId(next);
		setActiveProjectById(next === PLAYGROUND_SEEDED_KEY ? undefined : next);
	};

	const settingsProps = {
		model,
		models,
		modelsError,
		projectId,
		projects,
		keyStatus,
		keyError,
		systemPrompt,
		temperature,
		topP,
		maxTokens,
		responseFormat,
		jsonSchemaName,
		jsonSchemaStrict,
		jsonSchemaText,
		jsonSchemaError,
		onModelChange: setModel,
		onProjectChange,
		onSystemPromptChange: setSystemPrompt,
		onTemperatureChange: setTemperature,
		onTopPChange: setTopP,
		onMaxTokensChange: setMaxTokens,
		onResponseFormatChange: setResponseFormat,
		onJsonSchemaNameChange: setJsonSchemaName,
		onJsonSchemaStrictChange: setJsonSchemaStrict,
		onJsonSchemaTextChange: setJsonSchemaText,
	};

	const formatLabel =
		responseFormat === "text"
			? "text"
			: responseFormat === "json_object"
				? "json"
				: "schema";

	const composerDisabled =
		isBusy || !model || keyStatus !== "ready" || !resolvedApiKey;

	return (
		<PageBody className="min-h-0 flex-1 gap-3 overflow-hidden">
			<div className="flex shrink-0 items-start justify-between gap-3">
				<div className="min-w-0 space-y-1">
					<h1 className="text-2xl font-semibold tracking-tight">
						Chat playground
					</h1>
					<p className="truncate text-sm text-muted-foreground">
						{GATEWAY_API_URL}
						{model ? ` · ${model}` : ""} · {useStream ? "stream" : "non-stream"}{" "}
						· temp {temperature.toFixed(2)} · {formatLabel}
					</p>
				</div>
				<Sheet>
					<SheetTrigger
						render={
							<Button
								variant="outline"
								size="icon"
								className="lg:hidden"
								aria-label="Open settings"
							>
								<Settings2Icon />
							</Button>
						}
					/>
					<SheetContent side="right" className="w-full sm:max-w-md">
						<SheetHeader>
							<SheetTitle>Settings</SheetTitle>
							<SheetDescription>
								Model, generation params, and structured output for this
								session.
							</SheetDescription>
						</SheetHeader>
						<div className="overflow-y-auto px-4 pb-4">
							<SettingsFields {...settingsProps} />
						</div>
					</SheetContent>
				</Sheet>
			</div>

			<div className="flex min-h-0 flex-1 gap-4 overflow-hidden">
				<MessageScrollerProvider>
					<Card className="flex min-h-0 flex-1 flex-col gap-0 overflow-hidden">
						<CardHeader className="shrink-0 gap-1 border-b">
							<CardTitle>Conversation</CardTitle>
							<CardDescription className="truncate">
								OpenAI-compatible chat through the local gateway
							</CardDescription>
							<CardAction>
								<Tooltip>
									<TooltipTrigger
										render={
											<Button
												variant="outline"
												size="icon"
												aria-label="Reset conversation"
												disabled={messages.length === 0 && !error}
												onClick={() => {
													setMessages([]);
													setError(null);
												}}
											>
												<RotateCwIcon />
											</Button>
										}
									/>
									<TooltipContent>
										<p>Reset</p>
									</TooltipContent>
								</Tooltip>
							</CardAction>
						</CardHeader>

						<CardContent className="flex min-h-0 flex-1 flex-col overflow-hidden p-0">
							{messages.length === 0 ? (
								<Empty className="h-full min-h-0">
									<EmptyHeader>
										<EmptyMedia variant="icon">
											<MessageCircleDashedIcon />
										</EmptyMedia>
										<EmptyTitle>Send a message</EmptyTitle>
										<EmptyDescription>
											Traffic hits the gateway with your configured virtual API
											key. The composer stays pinned below.
										</EmptyDescription>
									</EmptyHeader>
								</Empty>
							) : (
								<MessageScroller className="min-h-0 flex-1">
									<MessageScrollerViewport>
										<MessageScrollerContent
											aria-busy={isBusy}
											className="p-(--card-spacing)"
										>
											{messages.map((message) => {
												const isEmptyAssistant =
													message.role === "assistant" &&
													!(message.parts[0]?.text ?? "");
												if (isEmptyAssistant && waitingForFirstToken) {
													return (
														<div
															key={message.id}
															className="flex items-center gap-2 text-sm text-muted-foreground"
														>
															<Loader2Icon className="size-4 animate-spin" />
															Thinking…
														</div>
													);
												}
												if (isEmptyAssistant) return null;
												return (
													<MessageAnimated
														key={message.id}
														message={message}
														scrollAnchor={message.role === "user"}
													/>
												);
											})}
										</MessageScrollerContent>
									</MessageScrollerViewport>
									<MessageScrollerButton />
								</MessageScroller>
							)}
						</CardContent>

						<CardFooter className="shrink-0 flex-col items-stretch gap-2">
							{error ? (
								<p className="text-xs text-destructive">{error}</p>
							) : null}
							<form
								onSubmit={(e) => {
									e.preventDefault();
									void send();
								}}
								className="w-full"
							>
								<InputGroup className="h-auto items-end py-1">
									<InputGroupTextarea
										ref={textareaRef}
										value={input}
										onChange={(e) => setInput(e.target.value)}
										placeholder="Message the gateway…"
										disabled={composerDisabled}
										rows={1}
										className="max-h-40 min-h-10 field-sizing-content"
										onKeyDown={(e) => {
											if (e.key === "Enter" && !e.shiftKey) {
												e.preventDefault();
												void send();
											}
										}}
									/>
									<InputGroupAddon align="inline-end" className="pr-1.5 pb-1">
										<InputGroupButton
											type="submit"
											variant="default"
											size="icon-sm"
											disabled={
												composerDisabled ||
												!input.trim() ||
												(responseFormat === "json_schema" &&
													Boolean(jsonSchemaError))
											}
											aria-label="Send"
										>
											{isBusy || keyStatus === "provisioning" ? (
												<Loader2Icon className="animate-spin" />
											) : (
												<ArrowUpIcon />
											)}
										</InputGroupButton>
									</InputGroupAddon>
								</InputGroup>
								<p className="mt-1.5 text-xs text-muted-foreground">
									Enter to send · Shift+Enter for a new line
								</p>
							</form>
						</CardFooter>
					</Card>
				</MessageScrollerProvider>

				<Card className="hidden w-96 shrink-0 overflow-y-auto lg:flex lg:flex-col">
					<CardHeader className="shrink-0">
						<CardTitle className="text-base">Settings</CardTitle>
						<CardDescription>
							Model, generation, and structured output
						</CardDescription>
					</CardHeader>
					<CardContent className="min-h-0">
						<SettingsFields {...settingsProps} />
					</CardContent>
				</Card>
			</div>
		</PageBody>
	);
}
