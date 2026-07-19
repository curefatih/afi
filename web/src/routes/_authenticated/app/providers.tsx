import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import {
	createProviderMutationOptions,
	deleteProviderMutationOptions,
	providersQueryOptions,
} from "#/api/provider";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/providers")({
	staticData: {
		getTitle: () => "Providers",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const org = useActiveOrg();
	const orgId = org?.id ?? "";
	const qc = useQueryClient();
	const providers = useQuery(providersQueryOptions(orgId));
	const create = useMutation({
		...createProviderMutationOptions(),
		onSuccess: () =>
			qc.invalidateQueries({ queryKey: ["organizations", orgId, "providers"] }),
	});
	const del = useMutation({
		...deleteProviderMutationOptions(),
		onSuccess: () =>
			qc.invalidateQueries({ queryKey: ["organizations", orgId, "providers"] }),
	});

	const [name, setName] = useState("OpenAI");
	const [type, setType] = useState("openai");
	const [baseURL, setBaseURL] = useState("https://api.openai.com/v1");
	const [apiKeyEnv, setApiKeyEnv] = useState("OPENAI_API_KEY");
	const [error, setError] = useState<string | null>(null);

	const applyTypeDefaults = (next: string) => {
		setType(next);
		if (next === "anthropic") {
			setName("Anthropic");
			setBaseURL("https://api.anthropic.com/v1");
			setApiKeyEnv("ANTHROPIC_API_KEY");
		} else if (next === "gemini") {
			setName("Gemini");
			setBaseURL("https://generativelanguage.googleapis.com/v1beta");
			setApiKeyEnv("GEMINI_API_KEY");
		} else if (next === "openai_compatible") {
			setName("Ollama / compatible");
			setBaseURL("http://127.0.0.1:11434/v1");
			setApiKeyEnv("OLLAMA_API_KEY");
		} else if (next === "openai") {
			setName("OpenAI");
			setBaseURL("https://api.openai.com/v1");
			setApiKeyEnv("OPENAI_API_KEY");
		}
	};

	return (
		<PageBody>
			<PageHeader
				title="Providers"
				description="Register upstream LLM providers. Changes publish a new gateway snapshot automatically."
			/>
			<QueryGate
				isPending={providers.isPending}
				isError={providers.isError}
				error={providers.error}
				onRetry={() => providers.refetch()}
			>
				<div className="grid gap-6 lg:grid-cols-2">
					<div className="space-y-3">
						<h3 className="text-sm font-medium">Configured</h3>
						<ul className="divide-y rounded-md border">
							{(providers.data ?? []).map((p) => (
								<li
									key={p.id}
									className="flex items-start justify-between gap-2 p-3 text-sm"
								>
									<div>
										<div className="font-medium">{p.name}</div>
										<div className="text-muted-foreground">
											{p.type} · {p.base_url}
										</div>
										<div className="text-muted-foreground text-xs">
											env: {p.api_key_env}
											{p.capabilities?.stream
												? " · stream"
												: " · no-stream"}
										</div>
									</div>
									<Button
										variant="outline"
										size="sm"
										disabled={del.isPending}
										onClick={() => del.mutate(p.id)}
									>
										Delete
									</Button>
								</li>
							))}
							{(providers.data ?? []).length === 0 ? (
								<li className="text-muted-foreground p-3 text-sm">
									No providers yet.
								</li>
							) : null}
						</ul>
					</div>
					<form
						className="space-y-3 rounded-md border p-4"
						onSubmit={(e) => {
							e.preventDefault();
							if (!orgId) return;
							setError(null);
							create.mutate(
								{
									orgId,
									name,
									base_url: baseURL,
									api_key_env: apiKeyEnv,
									type,
								},
								{
									onError: (err) =>
										setError(
											err instanceof Error ? err.message : "Create failed",
										),
								},
							);
						}}
					>
						<h3 className="text-sm font-medium">Add provider</h3>
						<div className="space-y-1">
							<Label htmlFor="prov-type">Type</Label>
							<select
								id="prov-type"
								className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
								value={type}
								onChange={(e) => applyTypeDefaults(e.target.value)}
							>
								<option value="openai">openai</option>
								<option value="anthropic">anthropic</option>
								<option value="gemini">gemini</option>
								<option value="openai_compatible">
									openai_compatible
								</option>
							</select>
						</div>
						<div className="space-y-1">
							<Label htmlFor="prov-name">Name</Label>
							<Input
								id="prov-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="prov-base">Base URL</Label>
							<Input
								id="prov-base"
								value={baseURL}
								onChange={(e) => setBaseURL(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="prov-env">API key env var</Label>
							<Input
								id="prov-env"
								value={apiKeyEnv}
								onChange={(e) => setApiKeyEnv(e.target.value)}
								required
							/>
						</div>
						{error ? (
							<p className="text-destructive text-xs">{error}</p>
						) : null}
						<Button type="submit" disabled={create.isPending || !orgId}>
							Create & publish
						</Button>
					</form>
				</div>
			</QueryGate>
		</PageBody>
	);
}
