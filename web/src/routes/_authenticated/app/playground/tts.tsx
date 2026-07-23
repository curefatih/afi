import { createFileRoute, Link } from "@tanstack/react-router";
import { Loader2Icon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { PageBody, PageHeader } from "#/components/page-header";
import { BowlAudioPlayer } from "#/components/playground/bowl-audio-player";
import { MagicalBowl } from "#/components/playground/magical-bowl";
import { Button } from "#/components/ui/button";
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

export const Route = createFileRoute("/_authenticated/app/playground/tts")({
	...pageTitle("TTS"),
	component: RouteComponent,
});

const VOICES = ["alloy", "echo", "fable", "onyx", "nova", "shimmer"] as const;

function RouteComponent() {
	const [models, setModels] = useState<GatewayModel[]>([]);
	const [model, setModel] = useState("");
	const [voice, setVoice] = useState<string>("alloy");
	const [text, setText] = useState("Hello from AFI.");
	const [audioUrl, setAudioUrl] = useState<string | null>(null);
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [modelsError, setModelsError] = useState<string | null>(null);
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
				const list = (data.data ?? []).filter(isTTSModel);
				setModels(list);
				setModel((prev) => {
					if (list.some((m) => m.id === prev)) return prev;
					return list.find((m) => m.id === "tts-1")?.id ?? list[0]?.id ?? "";
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
				body: JSON.stringify({
					model,
					input,
					voice,
				}),
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
				description="OpenAI-compatible TTS via the gateway."
				info="Lists catalog models with mode audio_speech (e.g. tts-1)."
			/>
			<div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(280px,360px)]">
				<div className="space-y-5">
					{modelsError ? (
						<p className="text-destructive text-sm">{modelsError}</p>
					) : null}
					{models.length === 0 && !modelsError ? (
						<p className="text-muted-foreground text-sm">
							No TTS routes. Add <code className="text-xs">tts-1</code> under{" "}
							<Link to="/app/routing" className="underline">
								Routing
							</Link>{" "}
							or run <code className="text-xs">make seed</code>.
						</p>
					) : null}
					<div className="grid gap-4 sm:grid-cols-2">
						<div className="space-y-1.5">
							<Label>Model</Label>
							<Select value={model} onValueChange={(v) => setModel(v ?? "")}>
								<SelectTrigger className="w-full">
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
							<Label>Voice</Label>
							<Select
								value={voice}
								onValueChange={(v) => setVoice(v ?? "alloy")}
							>
								<SelectTrigger className="w-full">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									{VOICES.map((v) => (
										<SelectItem key={v} value={v}>
											{v}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="tts-text">Text</Label>
						<Textarea
							id="tts-text"
							value={text}
							onChange={(e) => setText(e.target.value)}
							rows={10}
							className="min-h-48 text-base"
						/>
					</div>
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
