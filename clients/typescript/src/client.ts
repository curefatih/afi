/** Options for {@link PlatformClient}. */
export type PlatformClientOptions = {
	/** Control plane base URL, e.g. `http://localhost:8081`. */
	baseUrl: string;
	/** Return a JWT for authenticated calls. Omit / return null for public endpoints. */
	getToken?: () => string | null | undefined | Promise<string | null | undefined>;
	/** Optional custom fetch (defaults to global fetch). */
	fetch?: typeof fetch;
};

export class PlatformApiError extends Error {
	readonly status: number;
	readonly body: unknown;

	constructor(message: string, status: number, body?: unknown) {
		super(message);
		this.name = "PlatformApiError";
		this.status = status;
		this.body = body;
	}
}

type RequestOpts = {
	method?: string;
	body?: unknown;
	auth?: boolean;
	query?: Record<string, string | number | boolean | undefined | null>;
};

function buildQuery(query?: RequestOpts["query"]): string {
	if (!query) return "";
	const params = new URLSearchParams();
	for (const [k, v] of Object.entries(query)) {
		if (v === undefined || v === null) continue;
		params.set(k, String(v));
	}
	const qs = params.toString();
	return qs ? `?${qs}` : "";
}

/**
 * Thin fetch client for `/api/v1/platform/*`.
 *
 * Types for responses live in `schema.gen.ts` (from `make openapi-gen`).
 */
export class PlatformClient {
	readonly baseUrl: string;
	private readonly getToken?: PlatformClientOptions["getToken"];
	private readonly fetchImpl: typeof fetch;

	constructor(opts: PlatformClientOptions) {
		this.baseUrl = opts.baseUrl.replace(/\/$/, "");
		this.getToken = opts.getToken;
		this.fetchImpl = opts.fetch ?? fetch;
	}

	async request<T>(path: string, opts: RequestOpts = {}): Promise<T> {
		const { method = "GET", body, auth = true, query } = opts;
		const headers = new Headers();
		if (body !== undefined) {
			headers.set("Content-Type", "application/json");
		}
		if (auth) {
			const token = this.getToken ? await this.getToken() : undefined;
			if (!token) {
				throw new PlatformApiError("missing access token", 401);
			}
			headers.set("Authorization", `Bearer ${token}`);
		}
		const res = await this.fetchImpl(
			`${this.baseUrl}${path}${buildQuery(query)}`,
			{
				method,
				headers,
				body: body === undefined ? undefined : JSON.stringify(body),
			},
		);
		if (res.status === 204) {
			return undefined as T;
		}
		const text = await res.text();
		let parsed: unknown = undefined;
		if (text) {
			try {
				parsed = JSON.parse(text);
			} catch {
				parsed = text;
			}
		}
		if (!res.ok) {
			let message = res.statusText || "request failed";
			if (
				parsed &&
				typeof parsed === "object" &&
				"error" in parsed &&
				typeof (parsed as { error: unknown }).error === "string"
			) {
				message = (parsed as { error: string }).error;
			}
			throw new PlatformApiError(message, res.status, parsed);
		}
		return parsed as T;
	}

	healthz() {
		return this.request<Record<string, unknown>>("/healthz", { auth: false });
	}

	login(body: { email: string; password: string }) {
		return this.request<{ token: string }>("/api/v1/platform/auth/login", {
			method: "POST",
			body,
			auth: false,
		});
	}

	me() {
		return this.request<{
			id: string;
			name: string;
			email: string;
			role: string;
		}>("/api/v1/platform/auth/me");
	}

	listOrganizations() {
		return this.request<Array<{ id: string; name: string; created_at?: string }>>(
			"/api/v1/platform/organizations",
		);
	}

	listRegions() {
		return this.request<
			Array<{
				id: string;
				slug: string;
				name: string;
				status: string;
				created_at?: string;
				updated_at?: string;
			}>
		>("/api/v1/platform/regions");
	}

	createRegion(body: { slug: string; name: string }) {
		return this.request<{
			id: string;
			slug: string;
			name: string;
			status: string;
		}>("/api/v1/platform/regions", { method: "POST", body });
	}

	listDeployments(regionID: string) {
		return this.request<unknown[]>(
			`/api/v1/platform/regions/${encodeURIComponent(regionID)}/deployments`,
		);
	}

	registerDeployment(
		regionID: string,
		body: { name: string; public_base_url?: string },
	) {
		return this.request<{ deployment: unknown; join_token: string }>(
			`/api/v1/platform/regions/${encodeURIComponent(regionID)}/deployments`,
			{ method: "POST", body },
		);
	}

	listRegionMemberships(regionID: string) {
		return this.request<
			Array<{
				organization_id: string;
				region_id: string;
				status: string;
				created_at?: string;
				updated_at?: string;
			}>
		>(
			`/api/v1/platform/regions/${encodeURIComponent(regionID)}/organizations`,
		);
	}

	bindOrgToRegion(
		regionID: string,
		body: { organization_id: string; status?: string },
	) {
		return this.request<{
			organization_id: string;
			region_id: string;
			status: string;
		}>(
			`/api/v1/platform/regions/${encodeURIComponent(regionID)}/organizations`,
			{ method: "POST", body },
		);
	}

	bindAllOrgsToRegion(regionID: string) {
		return this.request<{ bound: number }>(
			`/api/v1/platform/regions/${encodeURIComponent(regionID)}/organizations/bind-all`,
			{ method: "POST" },
		);
	}

	unbindOrgFromRegion(regionID: string, orgID: string) {
		return this.request<void>(
			`/api/v1/platform/regions/${encodeURIComponent(regionID)}/organizations/${encodeURIComponent(orgID)}`,
			{ method: "DELETE" },
		);
	}

	getRegionOverlay(regionID: string, orgID: string) {
		return this.request<{
			organization_id: string;
			region_id: string;
			payload: Record<string, unknown>;
		}>(
			`/api/v1/platform/regions/${encodeURIComponent(regionID)}/organizations/${encodeURIComponent(orgID)}/overlay`,
		);
	}

	putRegionOverlay(
		regionID: string,
		orgID: string,
		payload: Record<string, unknown>,
	) {
		return this.request<{
			organization_id: string;
			region_id: string;
			payload: Record<string, unknown>;
		}>(
			`/api/v1/platform/regions/${encodeURIComponent(regionID)}/organizations/${encodeURIComponent(orgID)}/overlay`,
			{ method: "PUT", body: payload },
		);
	}

	deleteRegionOverlay(regionID: string, orgID: string) {
		return this.request<void>(
			`/api/v1/platform/regions/${encodeURIComponent(regionID)}/organizations/${encodeURIComponent(orgID)}/overlay`,
			{ method: "DELETE" },
		);
	}

	listFederationPeers() {
		return this.request<
			Array<{
				id: string;
				name: string;
				region_id: string;
				base_url?: string;
				status: string;
				last_sync_at?: string | null;
				last_sync_cursor: number;
				last_sync_error?: string;
				created_at?: string;
				updated_at?: string;
			}>
		>("/api/v1/platform/federation/peers");
	}

	registerFederationPeer(body: {
		name: string;
		region_id: string;
		base_url?: string;
	}) {
		return this.request<{ peer: unknown; join_token: string }>(
			"/api/v1/platform/federation/peers",
			{ method: "POST", body },
		);
	}

	getFederationPeer(peerID: string) {
		return this.request<unknown>(
			`/api/v1/platform/federation/peers/${encodeURIComponent(peerID)}`,
		);
	}

	updateFederationPeer(
		peerID: string,
		body: { name?: string; base_url?: string; status?: string },
	) {
		return this.request<unknown>(
			`/api/v1/platform/federation/peers/${encodeURIComponent(peerID)}`,
			{ method: "PATCH", body },
		);
	}

	rotateFederationPeerJoinToken(peerID: string) {
		return this.request<{ peer: unknown; join_token: string }>(
			`/api/v1/platform/federation/peers/${encodeURIComponent(peerID)}/rotate-join-token`,
			{ method: "POST" },
		);
	}

	createOrganization(body: { name: string }) {
		return this.request<{ id: string; name: string; created_at?: string }>(
			"/api/v1/platform/organizations",
			{ method: "POST", body },
		);
	}

	listOrgKeys(orgID: string) {
		return this.request<unknown[]>(
			`/api/v1/platform/organizations/${encodeURIComponent(orgID)}/keys`,
		);
	}

	createOrgKey(
		orgID: string,
		body: {
			name: string;
			kind: "personal" | "service_account";
			project_id?: string;
			key?: string;
		},
	) {
		return this.request<unknown>(
			`/api/v1/platform/organizations/${encodeURIComponent(orgID)}/keys`,
			{ method: "POST", body },
		);
	}

	listProviders(orgID: string) {
		return this.request<unknown[]>(
			`/api/v1/platform/organizations/${encodeURIComponent(orgID)}/providers`,
		);
	}

	listRoutes(orgID: string) {
		return this.request<unknown[]>(
			`/api/v1/platform/organizations/${encodeURIComponent(orgID)}/routes`,
		);
	}

	listUsage(
		orgID: string,
		query?: Record<string, string | number | boolean | undefined | null>,
	) {
		return this.request<unknown[]>(
			`/api/v1/platform/organizations/${encodeURIComponent(orgID)}/usage`,
			{ query },
		);
	}

	listAudit(
		orgID: string,
		query?: Record<string, string | number | boolean | undefined | null>,
	) {
		return this.request<unknown[]>(
			`/api/v1/platform/organizations/${encodeURIComponent(orgID)}/audit`,
			{ query },
		);
	}
}
