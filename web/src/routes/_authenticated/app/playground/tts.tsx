import { createFileRoute, Link } from "@tanstack/react-router";
import { Loader2Icon } from "lucide-react";
import { useEffect, useState } from "react";
import { PageBody, PageHeader } from "#/components/page-header";
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

export const Route = createFileRoute("/_authenticated/app/playground/tts")({
	staticData: {
		getTitle: () => "TTS",
	},
	component: RouteComponent,
});

type GatewayModel = {
	id: string;
	supports_tts?: boolean;
};

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
				const list = (data.data ?? []).filter((m) => m.supports_tts);
				setModels(list);
				setModel((prev) => {
					if (list.some((m) => m.id === prev)) return prev;
					return (
						list.find((m) => m.id === "tts-1")?.id ?? list[0]?.id ?? ""
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
			if (audioUrl) URL.revokeObjectURL(audioUrl);
		};
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
				description="OpenAI-compatible TTS via the gateway. Use a TTS route such as tts-1 (seeded)."
			/>
			<div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(280px,360px)]">
				<div className="space-y-5">
					{modelsError ? (
						<p className="text-destructive text-sm">{modelsError}</p>
					) : null}
					{models.length === 0 && !modelsError ? (
						<p className="text-muted-foreground text-sm">
							No TTS routes. Add{" "}
							<code className="text-xs">tts-1</code> under{" "}
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
				<div className="bg-muted/30 space-y-3 rounded-xl border p-5">
					<h3 className="text-sm font-medium">Preview</h3>
					<p className="text-muted-foreground text-sm">
						Audio plays here after a successful generate.
					</p>
					{audioUrl ? (
						<audio controls src={audioUrl} className="w-full">
							<track kind="captions" />
						</audio>
					) : (
						<div className="text-muted-foreground flex min-h-32 items-center justify-center rounded-lg border border-dashed text-sm">
							No audio yet
						</div>
					)}
				</div>
			</div>
		</PageBody>
	);
}
