import { createFileRoute, Link } from "@tanstack/react-router";
import { ChevronRightIcon, Loader2Icon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { PageBody, PageHeader } from "#/components/page-header";
import { BowlAudioPlayer } from "#/components/playground/bowl-audio-player";
import { MagicalBowl } from "#/components/playground/magical-bowl";
import { Button } from "#/components/ui/button";
import {
	Collapsible,
	CollapsibleContent,
	CollapsibleTrigger,
} from "#/components/ui/collapsible";
import {
	Combobox,
	ComboboxCollection,
	ComboboxContent,
	ComboboxEmpty,
	ComboboxInput,
	ComboboxItem,
	ComboboxList,
} from "#/components/ui/combobox";
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
import { Textarea } from "#/components/ui/textarea";
import { GATEWAY_API_KEY, GATEWAY_API_URL } from "#/lib/gateway-base";
import { type GatewayModel, isTTSModel } from "#/lib/gateway-models";
import { pageTitle } from "#/lib/page-meta";
import {
	EMPTY_EXTRA_JSON,
	type PlaygroundVoice,
	TTS_RESPONSE_FORMATS,
	buildTTSRequestBody,
	defaultVoiceForModel,
	pickPreferredModel,
	voicePresetsForModel,
} from "#/lib/playground-audio";

export const Route = createFileRoute("/_authenticated/app/playground/tts")({
	...pageTitle("TTS"),
	component: RouteComponent,
});

function RouteComponent() {
	const [models, setModels] = useState<GatewayModel[]>([]);
	const [model, setModel] = useState("");
	const [voice, setVoice] = useState("");
	const [responseFormat, setResponseFormat] = useState("");
	const [speed, setSpeed] = useState("");
	const [extraJSON, setExtraJSON] = useState(EMPTY_EXTRA_JSON);
	const [text, setText] = useState("Hello from AFI.");
	const [audioUrl, setAudioUrl] = useState<string | null>(null);
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [modelsError, setModelsError] = useState<string | null>(null);
	const audioRef = useRef<HTMLAudioElement>(null);
	const voiceTouchedRef = useRef(false);

	const selectedModel = useMemo(
		() => models.find((m) => m.id === model),
		[models, model],
	);
	const voicePresets = voicePresetsForModel(selectedModel);
	const voiceOptions = useMemo(() => {
		const opts: PlaygroundVoice[] = [...voicePresets];
		const custom = voice.trim();
		if (
			custom &&
			!opts.some(
				(p) =>
					p.id === custom || p.label.toLowerCase() === custom.toLowerCase(),
			)
		) {
			opts.unshift({ id: custom, label: custom });
		}
		return opts;
	}, [voicePresets, voice]);
	const selectedVoice = useMemo(() => {
		const match = voiceOptions.find((p) => p.id === voice);
		if (match) return match;
		if (voice.trim()) return { id: voice, label: voice };
		return null;
	}, [voiceOptions, voice]);

	const setVoiceValue = (next: string) => {
		voiceTouchedRef.current = true;
		setVoice(next);
	};

	useEffect(() => {
		let cancelled = false;
		(async () => {
			try {
				const res = await fetch(`${GATEWAY_API_URL}/v1/models`, {
					headers: { Authorization: `Bearer ${GATEWAY_API_KEY}` },
				});
				if (!res.ok) throw new Error(`models HTTP ${res.status}`);
				const data = (await res.json()) as { data?: GatewayModel[] };
				if (cancelled) return;
				const list = (data.data ?? []).filter(isTTSModel);
				setModels(list);
				setModel((prev) => {
					if (list.some((m) => m.id === prev)) return prev;
					return pickPreferredModel(list, ["tts-1", "tts-1-hd"]);
				});
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

	useEffect(() => {
		const next = models.find((m) => m.id === model);
		if (!next) return;
		if (!voiceTouchedRef.current) {
			setVoice(defaultVoiceForModel(next));
		}
	}, [model, models]);

	useEffect(() => {
		return () => {
			if (audioUrl) URL.revokeObjectURL(audioUrl);
		};
	}, [audioUrl]);

	useEffect(() => {
		const el = audioRef.current;
		if (!el || !audioUrl) return;
		el.load();
		void el.play().catch(() => {
			/* Autoplay may be blocked; user can press play. */
		});
	}, [audioUrl]);

	const generate = async () => {
		const input = text.trim();
		if (!input || !model || busy) return;
		const built = buildTTSRequestBody({
			model,
			input,
			voice,
			responseFormat,
			speed,
			extraJSON,
		});
		if (!built.ok) {
			setError(built.error);
			return;
		}
		setBusy(true);
		setError(null);
		if (audioUrl) {
			URL.revokeObjectURL(audioUrl);
			setAudioUrl(null);
		}
		try {
			const res = await fetch(`${GATEWAY_API_URL}/v1/audio/speech`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${GATEWAY_API_KEY}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify(built.body),
			});
			if (!res.ok) {
				const errBody = await res.text();
				throw new Error(errBody || `HTTP ${res.status}`);
			}
			const blob = await res.blob();
			setAudioUrl(URL.createObjectURL(blob));
		} catch (e) {
			setError(e instanceof Error ? e.message : "TTS failed");
		} finally {
			setBusy(false);
		}
	};

	return (
		<PageBody>
			<PageHeader
				title="Text to speech"
				description="Call gateway /v1/audio/speech with any routed TTS model."
				info="Pick a route from /v1/models (mode audio_speech). Use defaults for common providers, or set any OpenAI-compatible speech fields — including extras in Advanced."
			/>
			<div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(280px,360px)]">
				<div className="space-y-5">
					{modelsError ? (
						<p className="text-destructive text-sm">{modelsError}</p>
					) : null}
					{models.length === 0 && !modelsError ? (
						<p className="text-muted-foreground text-sm">
							No TTS routes yet. Add a speech route under{" "}
							<Link to="/app/routing" className="underline">
								Routing
							</Link>{" "}
							(any provider with TTS), then refresh.
						</p>
					) : null}

					<div className="space-y-1.5">
						<Label>Model</Label>
						<Select
							value={model}
							onValueChange={(v) => {
								voiceTouchedRef.current = false;
								setModel(v ?? "");
							}}
						>
							<SelectTrigger className="w-full">
								<SelectValue placeholder="Select model" />
							</SelectTrigger>
							<SelectContent>
								{models.map((m) => (
									<SelectItem key={m.id} value={m.id}>
										{m.provider_type ? `${m.id} · ${m.provider_type}` : m.id}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					<div className="space-y-1.5">
						<Label htmlFor="tts-voice">Voice</Label>
						<Combobox
							items={voiceOptions}
							value={selectedVoice}
							onValueChange={(item) => setVoiceValue(item?.id ?? "")}
							onInputValueChange={(next) => setVoiceValue(next)}
							itemToStringLabel={(item) => item.id}
							isItemEqualToValue={(a, b) => a.id === b.id}
						>
							<ComboboxInput
								id="tts-voice"
								placeholder={
									voicePresets.length > 0
										? "Select or type a voice id…"
										: "Type a voice name or id…"
								}
								className="w-full font-mono text-sm"
								showClear={!!voice}
							/>
							<ComboboxContent
								align="start"
								className="w-[var(--anchor-width)]"
							>
								<ComboboxEmpty>
									{voice.trim()
										? `Press Enter or select “${voice.trim()}”`
										: "Type a voice id"}
								</ComboboxEmpty>
								<ComboboxList>
									<ComboboxCollection>
										{(item: PlaygroundVoice) => {
											const isCustom =
												!voicePresets.some((p) => p.id === item.id) &&
												item.id === voice.trim();
											return (
												<ComboboxItem key={item.id} value={item}>
													{isCustom ? (
														<span>
															Use <span className="font-mono">{item.id}</span>
														</span>
													) : (
														<>
															<span>{item.label}</span>
															{item.label !== item.id ? (
																<span className="text-muted-foreground font-mono text-xs">
																	{item.id}
																</span>
															) : null}
														</>
													)}
												</ComboboxItem>
											);
										}}
									</ComboboxCollection>
								</ComboboxList>
							</ComboboxContent>
						</Combobox>
						<p className="text-muted-foreground text-xs">
							{voicePresets.length > 0
								? `Presets for ${selectedModel?.provider_type ?? "this provider"}; type any other voice id to use it.`
								: "No curated presets for this provider — enter any voice the upstream accepts."}
						</p>
					</div>

					<div className="grid gap-4 sm:grid-cols-2">
						<div className="space-y-1.5">
							<Label>Response format</Label>
							<Select
								value={responseFormat || "__default__"}
								onValueChange={(v) =>
									setResponseFormat(!v || v === "__default__" ? "" : v)
								}
							>
								<SelectTrigger className="w-full">
									<SelectValue placeholder="Provider default" />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="__default__">Provider default</SelectItem>
									{TTS_RESPONSE_FORMATS.filter(Boolean).map((f) => (
										<SelectItem key={f} value={f}>
											{f}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1.5">
							<Label htmlFor="tts-speed">Speed (optional)</Label>
							<Input
								id="tts-speed"
								value={speed}
								onChange={(e) => setSpeed(e.target.value)}
								placeholder="e.g. 1.0"
								inputMode="decimal"
								className="font-mono text-sm"
							/>
						</div>
					</div>

					<div className="space-y-1.5">
						<Label htmlFor="tts-text">Text</Label>
						<Textarea
							id="tts-text"
							value={text}
							onChange={(e) => setText(e.target.value)}
							rows={8}
							className="min-h-40 text-base"
						/>
					</div>

					<section className="rounded-md border">
						<Collapsible defaultOpen={false} className="group/collapsible">
							<div className="p-3">
								<CollapsibleTrigger className="flex w-full items-start gap-2 text-left">
									<ChevronRightIcon className="mt-0.5 size-4 shrink-0 transition-transform duration-200 group-data-open/collapsible:rotate-90" />
									<div>
										<p className="text-sm font-medium">Advanced config</p>
										<p className="text-muted-foreground text-xs">
											JSON object merged into the speech request. Form fields
											above override the same keys.
										</p>
									</div>
								</CollapsibleTrigger>
							</div>
							<CollapsibleContent className="space-y-2 border-t px-3 pb-3 pt-3">
								<JsonCodeEditor
									id="tts-extra"
									value={extraJSON}
									onChange={setExtraJSON}
									minHeight="8rem"
									placeholder='{"instructions":"Speak slowly"}'
								/>
							</CollapsibleContent>
						</Collapsible>
					</section>

					{error ? (
						<pre className="text-destructive max-h-40 overflow-auto rounded-md border border-destructive/30 bg-destructive/5 p-3 text-xs whitespace-pre-wrap">
							{error}
						</pre>
					) : null}
					<Button
						size="lg"
						onClick={() => void generate()}
						disabled={busy || !text.trim() || !model}
					>
						{busy ? (
							<>
								<Loader2Icon className="animate-spin" />
								Generating…
							</>
						) : (
							"Generate speech"
						)}
					</Button>
				</div>
				<div className="bg-muted/30 relative flex flex-col space-y-4 overflow-hidden rounded-xl border p-5">
					<div
						aria-hidden
						className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_50%_30%,oklch(0.7_0_0/0.12),transparent_65%)] dark:bg-[radial-gradient(ellipse_at_50%_30%,oklch(1_0_0/0.08),transparent_65%)]"
					/>
					<div className="relative order-1 space-y-1">
						<h3 className="text-sm font-medium">Preview</h3>
						<p className="text-muted-foreground text-sm">
							{busy
								? "The bowl gathers a reply…"
								: audioUrl
									? "Play to hear the voice — the bowl answers with its shape."
									: "Generate speech and the bowl will answer."}
						</p>
					</div>
					<audio
						ref={audioRef}
						src={audioUrl ?? undefined}
						className="sr-only"
						preload="auto"
						playsInline
					>
						<track kind="captions" />
					</audio>
					<MagicalBowl
						audioRef={audioRef}
						ready={!!audioUrl}
						busy={busy}
						className="relative order-2"
					/>
					<BowlAudioPlayer
						audioRef={audioRef}
						src={audioUrl}
						busy={busy}
						className="relative order-3"
					/>
				</div>
			</div>
		</PageBody>
	);
}
