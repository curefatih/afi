import { PLATFORM_API_URL } from "#/lib/api-base";
import { useAuthStore } from "#/state/auth-state";

export class ApiError extends Error {
	status: number;
	body: unknown;

	constructor(message: string, status: number, body?: unknown) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.body = body;
	}
}

type RequestOptions = Omit<RequestInit, "body"> & {
	body?: unknown;
	auth?: boolean;
};

async function parseError(res: Response): Promise<ApiError> {
	let body: unknown;
	let message = res.statusText || "Request failed";
	try {
		body = await res.json();
		if (
			body &&
			typeof body === "object" &&
			"error" in body &&
			typeof (body as { error: unknown }).error === "string"
		) {
			message = (body as { error: string }).error;
		}
	} catch {
		// ignore non-JSON error bodies
	}
	return new ApiError(message, res.status, body);
}

export async function apiFetch<T>(
	path: string,
	options: RequestOptions = {},
): Promise<T> {
	const { body, auth = true, headers: initHeaders, ...rest } = options;
	const headers = new Headers(initHeaders);

	if (body !== undefined && !headers.has("Content-Type")) {
		headers.set("Content-Type", "application/json");
	}

	if (auth) {
		const token = useAuthStore.getState().user?.accessToken;
		if (!token) {
			throw new ApiError("Not authenticated", 401);
		}
		headers.set("Authorization", `Bearer ${token}`);
	}

	const res = await fetch(`${PLATFORM_API_URL}${path}`, {
		...rest,
		headers,
		body: body === undefined ? undefined : JSON.stringify(body),
	});

	if (!res.ok) {
		throw await parseError(res);
	}

	if (res.status === 204) {
		return undefined as T;
	}

	return (await res.json()) as T;
}
