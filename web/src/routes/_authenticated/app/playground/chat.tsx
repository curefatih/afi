import { createFileRoute } from "@tanstack/react-router";
import {
	ArrowUpIcon,
	MessageCircleDashedIcon,
	RotateCwIcon,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { PageBody, PageHeader } from "#/components/page-header";
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
	InputGroupInput,
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
	content: Array<{
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

	const projects = activeOrg?.projects ?? [];

	const apiKey = useMemo(() => {
		return GATEWAY_API_KEY;
	}, []);

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
			content: [{ type: "text", text }],
		};
		const assistantId = crypto.randomUUID();
		const next = [...messages, userMsg];
		setMessages([
			...next,
			{
				id: assistantId,
				role: "assistant",
				content: [{ type: "text", text: "" }],
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
						content: m.content.map((c) => c.text).join("\n"),
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
									content: [
										{
											type: "text",
											text: (m.content[0]?.text ?? "") + piece,
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
						m.id !== assistantId || (m.content[0]?.text ?? "").length > 0,
				),
			);
		} finally {
			setIsBusy(false);
		}
	};

	return (
		<PageBody className="h-full min-h-0">
			<PageHeader
				title="Chat playground"
				description="Streams OpenAI-compatible chat through the local gateway. Models come from GET /v1/models (your org routes)."
			/>

			<div className="flex min-h-0 flex-1 flex-col gap-4 lg:flex-row">
				<div className="min-h-0 flex-1">
					<MessageScrollerProvider>
						<Card className="mx-auto h-full w-full gap-0">
							<CardHeader className="gap-1 border-b">
								<CardTitle>Conversation</CardTitle>
								<CardDescription>
									{GATEWAY_API_URL} · {model || "—"} · stream
								</CardDescription>
								<CardAction>
									<Tooltip>
										<TooltipTrigger
											render={
												<Button
													variant="outline"
													size="icon"
													aria-label="Reset conversation"
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
							<CardContent className="flex-1 overflow-hidden p-0">
								{messages.length === 0 ? (
									<Empty className="h-full">
										<EmptyHeader>
											<EmptyMedia variant="icon">
												<MessageCircleDashedIcon />
											</EmptyMedia>
											<EmptyTitle>Send a message</EmptyTitle>
											<EmptyDescription>
												Traffic hits the gateway with your configured virtual
												API key.
											</EmptyDescription>
										</EmptyHeader>
									</Empty>
								) : (
									<MessageScroller>
										<MessageScrollerViewport>
											<MessageScrollerContent
												aria-busy={isBusy}
												className="p-(--card-spacing)"
											>
												{messages.map((message) => (
													<MessageAnimated
														key={message.id}
														message={message}
														scrollAnchor={message.role === "user"}
													/>
												))}
											</MessageScrollerContent>
										</MessageScrollerViewport>
										<MessageScrollerButton />
									</MessageScroller>
								)}
							</CardContent>
							<CardFooter className="flex-col gap-2">
								{error ? (
									<p className="w-full text-xs text-destructive">{error}</p>
								) : null}
								<form
									onSubmit={(e) => {
										e.preventDefault();
										void send();
									}}
									className="w-full"
								>
									<InputGroup>
										<InputGroupInput
											value={input}
											onChange={(e) => setInput(e.target.value)}
											placeholder="Message the gateway…"
											disabled={isBusy || !model}
										/>
										<InputGroupAddon align="inline-end">
											<InputGroupButton
												type="submit"
												variant="default"
												size="icon-sm"
												disabled={isBusy || !input.trim() || !model}
											>
												<ArrowUpIcon />
												<span className="sr-only">Send</span>
											</InputGroupButton>
										</InputGroupAddon>
									</InputGroup>
								</form>
							</CardFooter>
						</Card>
					</MessageScrollerProvider>
				</div>

				<Card className="w-full shrink-0 lg:w-72">
					<CardHeader>
						<CardTitle className="text-base">Settings</CardTitle>
						<CardDescription>Model and auth context</CardDescription>
					</CardHeader>
					<CardContent className="space-y-4">
						<div className="space-y-2">
							<Label>Model</Label>
							{modelsError ? (
								<p className="text-destructive text-xs">{modelsError}</p>
							) : null}
							<Select
								value={model}
								onValueChange={(value) => setModel(value ?? models[0]?.id ?? "")}
								disabled={models.length === 0}
							>
								<SelectTrigger className="w-full">
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
							<Label>Project context</Label>
							<Select
								value={projectId}
								onValueChange={(value) => {
									const next = value ?? "seeded";
									setProjectId(next);
									setActiveProjectById(next === "seeded" ? undefined : next);
								}}
							>
								<SelectTrigger className="w-full">
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
							<p className="text-xs text-muted-foreground">
								Requests currently use the seeded gateway key. Create project
								keys under API Keys for production-like auth.
							</p>
						</div>

						<div className="space-y-2">
							<Label>Active key</Label>
							<code className="block break-all rounded-md border bg-muted/40 px-2 py-1.5 text-xs">
								{apiKey}
							</code>
						</div>
					</CardContent>
				</Card>
			</div>
		</PageBody>
	);
}
