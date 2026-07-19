import { createFileRoute } from "@tanstack/react-router";
import {
	ArrowUpIcon,
	Loader2Icon,
	MessageCircleDashedIcon,
	RotateCwIcon,
	Settings2Icon,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
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
import {
	InputGroup,
	InputGroupAddon,
	InputGroupButton,
	InputGroupTextarea,
} from "#/components/ui/input-group";
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
import {
	Sheet,
	SheetContent,
	SheetDescription,
	SheetHeader,
	SheetTitle,
	SheetTrigger,
} from "#/components/ui/sheet";
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "#/components/ui/tooltip";
import { GATEWAY_API_KEY, GATEWAY_API_URL } from "#/lib/gateway-base";
import { useActiveOrg, useOrgActions } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/playground/chat")({
	staticData: {
		getTitle: () => "Chat",
	},
	component: RouteComponent,
});

type GatewayModel = { id: string };

type Message = {
	id: string;
	role: "user" | "assistant";
	parts: Array<{
		type: "text";
		text: string;
	}>;
};

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

function SettingsFields({
	model,
	models,
	modelsError,
	projectId,
	projects,
	apiKey,
	onModelChange,
	onProjectChange,
}: {
	model: string;
	models: GatewayModel[];
	modelsError: string | null;
	projectId: string;
	projects: Array<{ id: string; name: string }>;
	apiKey: string;
	onModelChange: (value: string) => void;
	onProjectChange: (value: string) => void;
}) {
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
					From gateway /v1/models (org routes in the current snapshot).
				</p>
			</div>

			<div className="space-y-2">
				<Label htmlFor="chat-project">Project context</Label>
				<Select
					value={projectId}
					onValueChange={(value) => onProjectChange(value ?? "seeded")}
				>
					<SelectTrigger id="chat-project" className="w-full">
						<SelectValue placeholder="Seeded key" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="seeded">Seeded local key</SelectItem>
						{projects.map((project) => (
							<SelectItem key={project.id} value={project.id}>
								{project.name}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
				<p className="text-muted-foreground text-xs">
					Requests currently use the seeded gateway key. Create project keys
					under API Keys for production-like auth.
				</p>
			</div>

			<div className="space-y-2">
				<Label>Active key</Label>
				<code className="block break-all rounded-md border bg-muted/40 px-2 py-1.5 text-xs">
					{apiKey}
				</code>
			</div>
		</div>
	);
}

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const { setActiveProjectById } = useOrgActions();
	const [input, setInput] = useState("");
	const [models, setModels] = useState<GatewayModel[]>([]);
	const [model, setModel] = useState("");
	const [modelsError, setModelsError] = useState<string | null>(null);
	const [isBusy, setIsBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [messages, setMessages] = useState<Message[]>([]);
	const [projectId, setProjectId] = useState<string>("seeded");
	const textareaRef = useRef<HTMLTextAreaElement>(null);

	const projects = activeOrg?.projects ?? [];

	const apiKey = useMemo(() => {
		return GATEWAY_API_KEY;
	}, []);

	const waitingForFirstToken =
		isBusy &&
		messages.length > 0 &&
		messages[messages.length - 1]?.role === "assistant" &&
		!(messages[messages.length - 1]?.parts[0]?.text ?? "");

	useEffect(() => {
		let cancelled = false;
		(async () => {
			try {
				const res = await fetch(`${GATEWAY_API_URL}/v1/models`, {
					headers: { Authorization: `Bearer ${GATEWAY_API_KEY}` },
				});
				if (!res.ok) {
					throw new Error(`models HTTP ${res.status}`);
				}
				const data = (await res.json()) as {
					data?: GatewayModel[];
				};
				if (cancelled) return;
				const list = (data.data ?? []).filter((m) => m.id);
				setModels(list);
				setModel((prev) => prev || list[0]?.id || "");
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
	}, []);

	const send = async () => {
		const text = input.trim();
		if (!text || isBusy || !model) return;

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

		try {
			const res = await fetch(`${GATEWAY_API_URL}/v1/chat/completions`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${apiKey}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					model,
					stream: true,
					messages: next.map((m) => ({
						role: m.role,
						content: m.parts.map((c) => c.text).join("\n"),
					})),
				}),
			});
			if (!res.ok) {
				const body = await res.text();
				throw new Error(body || `HTTP ${res.status}`);
			}

			await readSSEContent(res, (piece) => {
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
			});
		} catch (e) {
			setError(e instanceof Error ? e.message : "Request failed");
			setMessages((prev) =>
				prev.filter(
					(m) =>
						m.id !== assistantId || (m.parts[0]?.text ?? "").length > 0,
				),
			);
		} finally {
			setIsBusy(false);
			textareaRef.current?.focus();
		}
	};

	const onProjectChange = (next: string) => {
		setProjectId(next);
		setActiveProjectById(next === "seeded" ? undefined : next);
	};

	const settingsProps = {
		model,
		models,
		modelsError,
		projectId,
		projects,
		apiKey,
		onModelChange: setModel,
		onProjectChange,
	};

	return (
		<PageBody className="min-h-0 flex-1 gap-3 overflow-hidden">
			<div className="flex shrink-0 items-start justify-between gap-3">
				<div className="min-w-0 space-y-1">
					<h1 className="text-2xl font-semibold tracking-tight">
						Chat playground
					</h1>
					<p className="truncate text-sm text-muted-foreground">
						{GATEWAY_API_URL}
						{model ? ` · ${model}` : ""} · stream
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
					<SheetContent side="right" className="w-full sm:max-w-sm">
						<SheetHeader>
							<SheetTitle>Settings</SheetTitle>
							<SheetDescription>
								Model and auth context for this session.
							</SheetDescription>
						</SheetHeader>
						<div className="px-4 pb-4">
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
										disabled={isBusy || !model}
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
											disabled={isBusy || !input.trim() || !model}
											aria-label="Send"
										>
											{isBusy ? (
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

				<Card className="hidden w-72 shrink-0 overflow-y-auto lg:flex lg:flex-col">
					<CardHeader className="shrink-0">
						<CardTitle className="text-base">Settings</CardTitle>
						<CardDescription>Model and auth context</CardDescription>
					</CardHeader>
					<CardContent className="min-h-0">
						<SettingsFields {...settingsProps} />
					</CardContent>
				</Card>
			</div>
		</PageBody>
	);
}
