import { createFileRoute, Link } from "@tanstack/react-router";
import {
	ChevronRightIcon,
	Loader2Icon,
	MicIcon,
	SquareIcon,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { PageBody, PageHeader } from "#/components/page-header";
import { BowlAudioPlayer } from "#/components/playground/bowl-audio-player";
import { Button } from "#/components/ui/button";
import {
	Collapsible,
	CollapsibleContent,
	CollapsibleTrigger,
} from "#/components/ui/collapsible";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "#/components/ui/tabs";
import { Textarea } from "#/components/ui/textarea";
import { GATEWAY_API_KEY, GATEWAY_API_URL } from "#/lib/gateway-base";
import { type GatewayModel, isSTTModel } from "#/lib/gateway-models";
import { pageTitle } from "#/lib/page-meta";
import {
	appendSTTFormFields,
	EMPTY_EXTRA_JSON,
	pickPreferredModel,
	STT_RESPONSE_FORMATS,
} from "#/lib/playground-audio";

export const Route = createFileRoute("/_authenticated/app/playground/stt")({
	...pageTitle("STT"),
	component: RouteComponent,
});

const RECORD_MIME_CANDIDATES = [
	"audio/webm;codecs=opus",
	"audio/webm",
	"audio/mp4",
	"audio/ogg;codecs=opus",
] as const;

function pickRecorderMimeType(): string | undefined {
	if (typeof MediaRecorder === "undefined") return undefined;
	return RECORD_MIME_CANDIDATES.find((t) => MediaRecorder.isTypeSupported(t));
}

function extensionForMime(mime: string): string {
	if (mime.includes("mp4") || mime.includes("m4a")) return "m4a";
	if (mime.includes("ogg")) return "ogg";
	return "webm";
}

function formatDuration(ms: number): string {
	const totalSec = Math.floor(ms / 1000);
	const m = Math.floor(totalSec / 60);
	const s = totalSec % 60;
	return `${m}:${s.toString().padStart(2, "0")}`;
}

function RouteComponent() {
	const [models, setModels] = useState<GatewayModel[]>([]);
	const [model, setModel] = useState("");
	const [language, setLanguage] = useState("");
	const [prompt, setPrompt] = useState("");
	const [responseFormat, setResponseFormat] = useState("");
	const [extraJSON, setExtraJSON] = useState(EMPTY_EXTRA_JSON);
	const [file, setFile] = useState<File | null>(null);
	const [previewUrl, setPreviewUrl] = useState<string | null>(null);
	const [transcript, setTranscript] = useState("");
	const [busy, setBusy] = useState(false);
	const [recording, setRecording] = useState(false);
	const [elapsedMs, setElapsedMs] = useState(0);
	const [error, setError] = useState<string | null>(null);
	const [modelsError, setModelsError] = useState<string | null>(null);
	const [micSupported] = useState(
		() =>
			typeof navigator !== "undefined" &&
			!!navigator.mediaDevices?.getUserMedia &&
			typeof MediaRecorder !== "undefined",
	);

	const mediaRecorderRef = useRef<MediaRecorder | null>(null);
	const streamRef = useRef<MediaStream | null>(null);
	const chunksRef = useRef<Blob[]>([]);
	const startedAtRef = useRef<number>(0);
	const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
	const fileInputRef = useRef<HTMLInputElement>(null);
	const audioRef = useRef<HTMLAudioElement>(null);

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
				const list = (data.data ?? []).filter(isSTTModel);
				setModels(list);
				setModel((prev) => {
					if (list.some((m) => m.id === prev)) return prev;
					return pickPreferredModel(list, ["whisper-1"]);
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
		return () => {
			if (timerRef.current) {
				clearInterval(timerRef.current);
				timerRef.current = null;
			}
			const recorder = mediaRecorderRef.current;
			if (recorder && recorder.state !== "inactive") {
				recorder.ondataavailable = null;
				recorder.onstop = null;
				recorder.stop();
			}
			for (const track of streamRef.current?.getTracks() ?? []) {
				track.stop();
			}
			streamRef.current = null;
		};
	}, []);

	useEffect(() => {
		return () => {
			if (previewUrl) URL.revokeObjectURL(previewUrl);
		};
	}, [previewUrl]);

	const stopTracks = () => {
		for (const track of streamRef.current?.getTracks() ?? []) {
			track.stop();
		}
		streamRef.current = null;
	};

	const clearTimer = () => {
		if (timerRef.current) {
			clearInterval(timerRef.current);
			timerRef.current = null;
		}
	};

	const setAudioFile = (next: File | null) => {
		audioRef.current?.pause();
		setFile(next);
		setPreviewUrl(next ? URL.createObjectURL(next) : null);
		if (!next && fileInputRef.current) {
			fileInputRef.current.value = "";
		}
	};

	const startRecording = async () => {
		if (!micSupported || recording || busy) return;
		audioRef.current?.pause();
		setError(null);
		try {
			const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
			streamRef.current = stream;
			const mimeType = pickRecorderMimeType();
			const recorder = mimeType
				? new MediaRecorder(stream, { mimeType })
				: new MediaRecorder(stream);
			chunksRef.current = [];
			recorder.ondataavailable = (e) => {
				if (e.data.size > 0) chunksRef.current.push(e.data);
			};
			recorder.onerror = () => {
				setError("Microphone recording failed");
				setRecording(false);
				clearTimer();
				stopTracks();
			};
			recorder.onstop = () => {
				clearTimer();
				stopTracks();
				const type = recorder.mimeType || mimeType || "audio/webm";
				const blob = new Blob(chunksRef.current, { type });
				chunksRef.current = [];
				mediaRecorderRef.current = null;
				setRecording(false);
				if (blob.size === 0) {
					setError("No audio captured — try again");
					return;
				}
				const name = `recording.${extensionForMime(type)}`;
				setAudioFile(new File([blob], name, { type }));
			};
			mediaRecorderRef.current = recorder;
			startedAtRef.current = Date.now();
			setElapsedMs(0);
			timerRef.current = setInterval(() => {
				setElapsedMs(Date.now() - startedAtRef.current);
			}, 200);
			recorder.start(250);
			setRecording(true);
		} catch (e) {
			stopTracks();
			const msg =
				e instanceof DOMException && e.name === "NotAllowedError"
					? "Microphone permission denied"
					: e instanceof Error
						? e.message
						: "Could not access microphone";
			setError(msg);
		}
	};

	const stopRecording = () => {
		const recorder = mediaRecorderRef.current;
		if (!recorder || recorder.state === "inactive") {
			setRecording(false);
			clearTimer();
			stopTracks();
			return;
		}
		recorder.stop();
	};

	const transcribe = async () => {
		if (!file || !model || busy || recording) return;
		const form = new FormData();
		form.append("file", file, file.name || "audio.webm");
		const fields = appendSTTFormFields(form, {
			model,
			language,
			prompt,
			responseFormat,
			extraJSON,
		});
		if (!fields.ok) {
			setError(fields.error);
			return;
		}
		setBusy(true);
		setError(null);
		setTranscript("");
		try {
			const res = await fetch(`${GATEWAY_API_URL}/v1/audio/transcriptions`, {
				method: "POST",
				headers: { Authorization: `Bearer ${GATEWAY_API_KEY}` },
				body: form,
			});
			if (!res.ok) {
				const errBody = await res.text();
				throw new Error(errBody || `HTTP ${res.status}`);
			}
			const ct = res.headers.get("Content-Type") ?? "";
			if (ct.includes("application/json")) {
				const data = (await res.json()) as { text?: string };
				setTranscript(
					typeof data.text === "string"
						? data.text
						: JSON.stringify(data, null, 2),
				);
			} else {
				setTranscript(await res.text());
			}
		} catch (e) {
			setError(e instanceof Error ? e.message : "STT failed");
		} finally {
			setBusy(false);
		}
	};

	return (
		<PageBody>
			<PageHeader
				title="Speech to text"
				description="Call gateway /v1/audio/transcriptions with any routed STT model."
				info="Pick a route from /v1/models (mode audio_transcription). Optional language/prompt/format apply to OpenAI-shaped APIs; use Advanced for any extra multipart fields."
			/>
			<div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(280px,1fr)]">
				<div className="space-y-5">
					{modelsError ? (
						<p className="text-destructive text-sm">{modelsError}</p>
					) : null}
					{models.length === 0 && !modelsError ? (
						<p className="text-muted-foreground text-sm">
							No STT routes yet. Add a transcription route under{" "}
							<Link to="/app/routing" className="underline">
								Routing
							</Link>{" "}
							(any provider with STT), then refresh.
						</p>
					) : null}

					<div className="space-y-1.5">
						<Label>Model</Label>
						<Select value={model} onValueChange={(v) => setModel(v ?? "")}>
							<SelectTrigger className="w-full max-w-md">
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

					<div className="grid gap-4 sm:grid-cols-2 max-w-2xl">
						<div className="space-y-1.5">
							<Label htmlFor="stt-language">Language (optional)</Label>
							<Input
								id="stt-language"
								value={language}
								onChange={(e) => setLanguage(e.target.value)}
								placeholder="e.g. en"
								className="font-mono text-sm"
							/>
						</div>
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
									{STT_RESPONSE_FORMATS.filter(Boolean).map((f) => (
										<SelectItem key={f} value={f}>
											{f}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
					</div>

					<div className="space-y-1.5 max-w-2xl">
						<Label htmlFor="stt-prompt">Prompt (optional)</Label>
						<Textarea
							id="stt-prompt"
							value={prompt}
							onChange={(e) => setPrompt(e.target.value)}
							placeholder="Optional vocabulary / style hints"
							rows={2}
							className="min-h-16 text-sm"
						/>
					</div>

					<div className="space-y-1.5">
						<Label>Audio</Label>
						<Tabs defaultValue="upload" className="max-w-md">
							<TabsList>
								<TabsTrigger value="upload">Upload</TabsTrigger>
								<TabsTrigger value="mic" disabled={!micSupported}>
									Microphone
								</TabsTrigger>
							</TabsList>
							<TabsContent value="upload" className="space-y-3 pt-3">
								<Input
									ref={fileInputRef}
									id="stt-file"
									type="file"
									accept="audio/*,.mp3,.wav,.m4a,.webm,.ogg,.flac"
									className="cursor-pointer"
									disabled={recording || busy}
									onChange={(e) => setAudioFile(e.target.files?.[0] ?? null)}
								/>
								<p className="text-muted-foreground text-sm">
									Any audio format your routed provider accepts.
								</p>
							</TabsContent>
							<TabsContent value="mic" className="space-y-3 pt-3">
								{!micSupported ? (
									<p className="text-muted-foreground text-sm">
										This browser does not support microphone recording.
									</p>
								) : (
									<>
										<div className="flex flex-wrap items-center gap-2">
											{recording ? (
												<Button
													variant="destructive"
													onClick={stopRecording}
													disabled={busy}
												>
													<SquareIcon />
													Stop · {formatDuration(elapsedMs)}
												</Button>
											) : (
												<Button
													variant="outline"
													onClick={() => void startRecording()}
													disabled={busy}
												>
													<MicIcon />
													Start recording
												</Button>
											)}
											{recording ? (
												<span className="text-destructive flex items-center gap-1.5 text-sm">
													<span className="size-2 animate-pulse rounded-full bg-destructive" />
													Recording
												</span>
											) : null}
										</div>
										<p className="text-muted-foreground text-sm">
											Speak into your mic, then stop and transcribe.
										</p>
									</>
								)}
							</TabsContent>
						</Tabs>
						{file ? (
							<div className="max-w-md space-y-3">
								<p className="text-muted-foreground text-sm">
									{file.name} · {(file.size / 1024).toFixed(1)} KB
								</p>
								<audio
									ref={audioRef}
									src={previewUrl ?? undefined}
									className="sr-only"
									preload="metadata"
									playsInline
								>
									<track kind="captions" />
								</audio>
								{previewUrl ? (
									<BowlAudioPlayer
										audioRef={audioRef}
										src={previewUrl}
										busy={busy || recording}
									/>
								) : null}
								{!recording ? (
									<Button
										variant="outline"
										size="sm"
										onClick={() => setAudioFile(null)}
										disabled={busy}
									>
										Clear audio
									</Button>
								) : null}
							</div>
						) : null}
					</div>

					<section className="rounded-md border max-w-2xl">
						<Collapsible defaultOpen={false} className="group/collapsible">
							<div className="p-3">
								<CollapsibleTrigger className="flex w-full items-start gap-2 text-left">
									<ChevronRightIcon className="mt-0.5 size-4 shrink-0 transition-transform duration-200 group-data-open/collapsible:rotate-90" />
									<div>
										<p className="text-sm font-medium">Advanced config</p>
										<p className="text-muted-foreground text-xs">
											JSON object appended as multipart fields. Form fields
											above override the same keys.
										</p>
									</div>
								</CollapsibleTrigger>
							</div>
							<CollapsibleContent className="space-y-2 border-t px-3 pb-3 pt-3">
								<JsonCodeEditor
									id="stt-extra"
									value={extraJSON}
									onChange={setExtraJSON}
									minHeight="8rem"
									placeholder='{"temperature":0}'
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
						onClick={() => void transcribe()}
						disabled={busy || recording || !file || !model}
					>
						{busy ? (
							<>
								<Loader2Icon className="animate-spin" />
								Transcribing…
							</>
						) : (
							"Transcribe"
						)}
					</Button>
				</div>
				<div className="bg-muted/30 space-y-3 rounded-xl border p-5">
					<h3 className="text-sm font-medium">Transcript</h3>
					{transcript ? (
						<div className="min-h-48 text-base leading-relaxed whitespace-pre-wrap">
							{transcript}
						</div>
					) : (
						<div className="text-muted-foreground flex min-h-48 items-center justify-center rounded-lg border border-dashed text-sm">
							Transcript appears here
						</div>
					)}
				</div>
			</div>
		</PageBody>
	);
}
