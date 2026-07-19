export type CelSymbol = {
	label: string;
	insert: string;
	detail: string;
	type: "root" | "field" | "keyword" | "operator" | "snippet";
	group: string;
};

export const CEL_VARIABLES: CelSymbol[] = [
	{
		label: "request",
		insert: "request",
		detail: "Incoming request context",
		type: "root",
		group: "Roots",
	},
	{
		label: "key",
		insert: "key",
		detail: "Authenticated virtual API key",
		type: "root",
		group: "Roots",
	},
	{
		label: "request.model",
		insert: "request.model",
		detail: "Model id from the client (string)",
		type: "field",
		group: "Request",
	},
	{
		label: "request.path",
		insert: "request.path",
		detail: "Gateway path, e.g. /v1/chat/completions",
		type: "field",
		group: "Request",
	},
	{
		label: "request.stream",
		insert: "request.stream",
		detail: "Whether the client requested streaming (bool)",
		type: "field",
		group: "Request",
	},
	{
		label: "key.id",
		insert: "key.id",
		detail: "API key id",
		type: "field",
		group: "Key",
	},
	{
		label: "key.name",
		insert: "key.name",
		detail: "API key display name",
		type: "field",
		group: "Key",
	},
	{
		label: "key.kind",
		insert: "key.kind",
		detail: '"personal" or "service_account"',
		type: "field",
		group: "Key",
	},
	{
		label: "key.organization_id",
		insert: "key.organization_id",
		detail: "Owning organization id",
		type: "field",
		group: "Key",
	},
	{
		label: "key.project_id",
		insert: "key.project_id",
		detail: "Project id (empty for personal keys)",
		type: "field",
		group: "Key",
	},
	{
		label: "key.owner_user_id",
		insert: "key.owner_user_id",
		detail: "Owner user id (personal keys only)",
		type: "field",
		group: "Key",
	},
];

export const CEL_OPERATORS: CelSymbol[] = [
	{
		label: "==",
		insert: " == ",
		detail: "Equals",
		type: "operator",
		group: "Operators",
	},
	{
		label: "!=",
		insert: " != ",
		detail: "Not equals",
		type: "operator",
		group: "Operators",
	},
	{
		label: "&&",
		insert: " && ",
		detail: "And",
		type: "operator",
		group: "Operators",
	},
	{
		label: "||",
		insert: " || ",
		detail: "Or",
		type: "operator",
		group: "Operators",
	},
	{
		label: "!",
		insert: "!",
		detail: "Not",
		type: "operator",
		group: "Operators",
	},
	{
		label: "in",
		insert: " in ",
		detail: 'Membership, e.g. request.model in ["a", "b"]',
		type: "operator",
		group: "Operators",
	},
	{
		label: "true",
		insert: "true",
		detail: "Boolean true",
		type: "keyword",
		group: "Literals",
	},
	{
		label: "false",
		insert: "false",
		detail: "Boolean false (deny all matching traffic)",
		type: "keyword",
		group: "Literals",
	},
];

export type CelExample = {
	title: string;
	description: string;
	expression: string;
};

export const CEL_EXAMPLES: CelExample[] = [
	{
		title: "Block a model",
		description: "Deny one model id; allow everything else.",
		expression: 'request.model != "blocked-model"',
	},
	{
		title: "Allowlist models",
		description: "Only these model ids may pass.",
		expression: 'request.model in ["echo-demo", "gpt-4o-mini"]',
	},
	{
		title: "No streaming",
		description: "Reject stream: true chat requests.",
		expression: "!request.stream",
	},
	{
		title: "Chat paths only",
		description: "Limit to OpenAI-style chat and Anthropic messages.",
		expression:
			'request.path == "/v1/chat/completions" || request.path == "/v1/messages"',
	},
	{
		title: "Personal keys only",
		description: "Service-account keys are denied.",
		expression: 'key.kind == "personal"',
	},
];

const ALL_COMPLETIONS: CelSymbol[] = [
	...CEL_VARIABLES,
	...CEL_OPERATORS.filter((o) => o.type === "keyword"),
];

/** Token just before the cursor for autocomplete filtering. */
export function completionContext(
	value: string,
	cursor: number,
): { prefix: string; start: number; afterDot: boolean } {
	const before = value.slice(0, cursor);
	const match = before.match(/([A-Za-z_][\w.]*)?$/);
	const token = match?.[1] ?? "";
	const start = cursor - token.length;
	const afterDot = token.includes(".");
	return { prefix: token, start, afterDot };
}

export function filterCompletions(
	prefix: string,
	afterDot: boolean,
): CelSymbol[] {
	const p = prefix.toLowerCase();
	let pool = ALL_COMPLETIONS;
	if (afterDot) {
		pool = CEL_VARIABLES.filter((s) => s.type === "field");
	} else if (p === "" || !p.includes(".")) {
		pool = [
			...CEL_VARIABLES.filter((s) => s.type === "root"),
			...CEL_VARIABLES.filter((s) => s.type === "field"),
			...CEL_OPERATORS.filter((s) => s.type === "keyword"),
		];
	}
	if (!p) return pool.slice(0, 12);
	return pool.filter(
		(s) =>
			s.label.toLowerCase().startsWith(p) ||
			s.label.toLowerCase().includes(p) ||
			s.insert.toLowerCase().startsWith(p),
	);
}

export function applyCompletion(
	value: string,
	cursor: number,
	item: CelSymbol,
): { next: string; cursor: number } {
	const { start } = completionContext(value, cursor);
	const next = value.slice(0, start) + item.insert + value.slice(cursor);
	return { next, cursor: start + item.insert.length };
}

export function insertAtCursor(
	value: string,
	cursor: number,
	text: string,
	selectionEnd = cursor,
): { next: string; cursor: number } {
	const next = value.slice(0, cursor) + text + value.slice(selectionEnd);
	return { next, cursor: cursor + text.length };
}
