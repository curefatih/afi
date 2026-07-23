import { describe, expect, it } from "vitest";
import { safeRedirectPath } from "./auth-redirect";

describe("safeRedirectPath", () => {
	it("returns in-app paths", () => {
		expect(safeRedirectPath("/app/organizations")).toBe("/app/organizations");
		expect(safeRedirectPath("/app/projects?x=1#y")).toBe("/app/projects?x=1#y");
	});

	it("rejects absolute and protocol-relative URLs", () => {
		expect(safeRedirectPath("http://localhost:3000/app/organizations")).toBe(
			"/app/dashboard",
		);
		expect(safeRedirectPath("//evil.example/phish")).toBe("/app/dashboard");
		expect(safeRedirectPath(undefined)).toBe("/app/dashboard");
	});
});
