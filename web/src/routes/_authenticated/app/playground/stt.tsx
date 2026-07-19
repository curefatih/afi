import { createFileRoute, Link } from "@tanstack/react-router";
import { Loader2Icon } from "lucide-react";
import { useEffect, useState } from "react";
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
import { GATEWAY_API_KEY, GATEWAY_API_URL } from "#/lib/gateway-base";
import { type GatewayModel, isSTTModel } from "#/lib/gateway-models";
import { pageTitle } from "#/lib/page-meta";

export const Route = createFileRoute("/_authenticated/app/playground/stt")({
	...pageTitle("STT"),
	component: RouteComponent,
});

function RouteComponent() {
	const [models, setModels] = useState<GatewayModel[]>([]);
	const [model, setModel] = useState("");
	const [file, setFile] = useState<File | null>(null);
	const [transcript, setTranscript] = useState("");
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [modelsError, setModelsError] = useState<string | null>(null);

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

	const transcribe = async () => {
		if (!file || !model || busy) return;
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
				description="OpenAI-compatible transcriptions via the gateway. Lists catalog models with mode audio_transcription (e.g. whisper-1)."
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
						<Label htmlFor="stt-file">Audio file</Label>
						<Input
							id="stt-file"
							type="file"
							accept="audio/*,.mp3,.wav,.m4a,.webm,.ogg,.flac"
							className="max-w-md cursor-pointer"
							onChange={(e) => setFile(e.target.files?.[0] ?? null)}
						/>
						{file ? (
							<p className="text-muted-foreground text-sm">
								{file.name} · {(file.size / 1024).toFixed(1)} KB
							</p>
						) : (
							<p className="text-muted-foreground text-sm">
								mp3, wav, m4a, webm, and other OpenAI-supported formats.
							</p>
						)}
					</div>
					{error ? (
						<pre className="text-destructive max-h-40 overflow-auto rounded-md border border-destructive/30 bg-destructive/5 p-3 text-xs whitespace-pre-wrap">
							{error}
						</pre>
					) : null}
					<Button
						size="lg"
						onClick={() => void transcribe()}
						disabled={busy || !file || !model}
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
