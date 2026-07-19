import { describe, expect, it } from "vitest";
import { loginFormSchema } from "./login-form.schema";

describe("loginFormSchema", () => {
	it("accepts valid credentials", () => {
		const parsed = loginFormSchema.parse({
			email: "admin@afi.local",
			password: "admin1",
		});
		expect(parsed.email).toBe("admin@afi.local");
		expect(parsed.password).toBe("admin1");
	});

	it("rejects invalid email and short password", () => {
		const result = loginFormSchema.safeParse({
			email: "not-an-email",
			password: "1234",
		});
		expect(result.success).toBe(false);
	});
});
