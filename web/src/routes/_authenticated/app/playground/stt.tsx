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

export const Route = createFileRoute("/_authenticated/app/playground/stt")({
	staticData: {
		getTitle: () => "STT",
	},
	component: RouteComponent,
});

type GatewayModel = {
	id: string;
	supports_stt?: boolean;
};

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
				const list = (data.data ?? []).filter((m) => m.supports_stt);
				setModels(list);
				setModel((prev) => prev || list[0]?.id || "whisper-1");
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
			form.append("model", model);
			form.append("file", file);
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
				description="OpenAI-compatible transcriptions via the gateway. Requires a routed model with supports_stt (seed includes whisper-1)."
			/>
			<div className="mx-auto max-w-xl space-y-4">
				{modelsError ? (
					<p className="text-destructive text-sm">{modelsError}</p>
				) : null}
				{models.length === 0 && !modelsError ? (
					<p className="text-muted-foreground text-sm">
						No STT-capable routes. Add a{" "}
						<code className="text-xs">whisper-1</code> route on an OpenAI
						provider in{" "}
						<Link to="/app/routing" className="underline">
							Routing
						</Link>
						, or re-seed.
					</p>
				) : null}
				<div className="space-y-1">
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
				<div className="space-y-1">
					<Label htmlFor="stt-file">Audio file</Label>
					<Input
						id="stt-file"
						type="file"
						accept="audio/*,.mp3,.wav,.m4a,.webm"
						onChange={(e) => setFile(e.target.files?.[0] ?? null)}
					/>
				</div>
				{error ? (
					<pre className="text-destructive max-h-32 overflow-auto text-xs whitespace-pre-wrap">
						{error}
					</pre>
				) : null}
				<Button
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
				{transcript ? (
					<div className="rounded-md border p-3 text-sm whitespace-pre-wrap">
						{transcript}
					</div>
				) : null}
			</div>
		</PageBody>
	);
}
