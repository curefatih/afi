import { createFileRoute, Link } from "@tanstack/react-router";
import { Loader2Icon, MicIcon, SquareIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { PageBody, PageHeader } from "#/components/page-header";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
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
import { type GatewayModel, isSTTModel } from "#/lib/gateway-models";
import { pageTitle } from "#/lib/page-meta";

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
					return (
						list.find((m) => m.id === "whisper-1")?.id ?? list[0]?.id ?? ""
					);
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
		setFile(next);
		setPreviewUrl(next ? URL.createObjectURL(next) : null);
		if (!next && fileInputRef.current) {
			fileInputRef.current.value = "";
		}
	};

	const startRecording = async () => {
		if (!micSupported || recording || busy) return;
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
		setBusy(true);
		setError(null);
		setTranscript("");
		try {
			const form = new FormData();
			form.append("file", file, file.name || "audio.webm");
			form.append("model", model);
			const res = await fetch(`${GATEWAY_API_URL}/v1/audio/transcriptions`, {
				method: "POST",
				headers: { Authorization: `Bearer ${GATEWAY_API_KEY}` },
				body: form,
			});
			if (!res.ok) {
				const errBody = await res.text();
				throw new Error(errBody || `HTTP ${res.status}`);
			}
			const data = (await res.json()) as { text?: string };
			setTranscript(data.text ?? JSON.stringify(data));
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
				description="OpenAI-compatible transcriptions via the gateway."
				info="Lists catalog models with mode audio_transcription (e.g. whisper-1). Upload a file or record from the microphone."
			/>
			<div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(280px,1fr)]">
				<div className="space-y-5">
					{modelsError ? (
						<p className="text-destructive text-sm">{modelsError}</p>
					) : null}
					{models.length === 0 && !modelsError ? (
						<p className="text-muted-foreground text-sm">
							No STT routes. Add <code className="text-xs">whisper-1</code>{" "}
							under{" "}
							<Link to="/app/routing" className="underline">
								Routing
							</Link>{" "}
							or run <code className="text-xs">make seed</code>.
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
										{m.id}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
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
									onChange={(e) =>
										setAudioFile(e.target.files?.[0] ?? null)
									}
								/>
								<p className="text-muted-foreground text-sm">
									mp3, wav, m4a, webm, and other OpenAI-supported formats.
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
							<div className="max-w-md space-y-2">
								<p className="text-muted-foreground text-sm">
									{file.name} · {(file.size / 1024).toFixed(1)} KB
								</p>
								{previewUrl ? (
									<audio
										controls
										src={previewUrl}
										className="h-10 w-full"
										preload="metadata"
									>
										<track kind="captions" />
									</audio>
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
