/** Safe in-app path for post-login redirect (rejects absolute / protocol-relative URLs). */
export function safeRedirectPath(
	redirect: string | undefined,
	fallback = "/app/dashboard",
): string {
	if (!redirect) {
		return fallback;
	}
	if (!redirect.startsWith("/") || redirect.startsWith("//")) {
		return fallback;
	}
	return redirect;
}
