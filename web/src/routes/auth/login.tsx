import { useForm } from "@tanstack/react-form";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
	createFileRoute,
	Link,
	useRouter,
	useSearch,
} from "@tanstack/react-router";
import { toast } from "sonner";
import {
	loginMutationOptions,
	ssoProvidersQueryOptions,
	startSSO,
} from "#/api/auth";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";
import {
	Field,
	FieldDescription,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import { safeRedirectPath } from "#/lib/auth-redirect";
import { pageTitle } from "#/lib/page-meta";
import { cn } from "#/lib/utils";
import { loginFormSchema } from "#/schemas/login-form.schema";

export const Route = createFileRoute("/auth/login")({
	...pageTitle("Login", {
		description: "Sign in to the AFI control plane.",
	}),
	validateSearch: (search: Record<string, unknown>) => ({
		redirect:
			typeof search.redirect === "string" ? search.redirect : undefined,
	}),
	component: RouteComponent,
});

function RouteComponent() {
	const router = useRouter();
	const search = useSearch({ from: "/auth/login" });

	const loginMutation = useMutation({
		...loginMutationOptions(),
	});
	const ssoProviders = useQuery(ssoProvidersQueryOptions());

	const form = useForm({
		validators: {
			onChange: loginFormSchema,
		},
		defaultValues: {
			email: "",
			password: "",
		},
		onSubmit: async (values) => {
			loginMutation.mutate(
				{
					email: values.value.email,
					password: values.value.password,
				},
				{
					onSuccess: () => {
						toast.success("Welcome back");
						router.history.push(safeRedirectPath(search.redirect));
					},
					onError: (error) => {
						toast.error(error.message || "Login failed");
					},
				},
			);
		},
	});

	const providers = ssoProviders.data ?? [];

	return (
		<div className={cn("flex flex-col gap-6")}>
			<Card>
				<CardHeader className="text-center">
					<CardTitle className="text-xl">Welcome back</CardTitle>
					<CardDescription>
						Sign in with your platform email and password
					</CardDescription>
				</CardHeader>
				<CardContent>
					<form
						onSubmit={(e) => {
							e.preventDefault();
							void form.handleSubmit();
						}}
					>
						<FieldGroup>
							<form.Field name="email">
								{(field) => (
									<Field id="email">
										<FieldLabel htmlFor="email">Email</FieldLabel>
										<Input
											id="email"
											type="email"
											autoComplete="email"
											placeholder="admin@afi.local"
											value={field.state.value}
											onChange={(e) => field.handleChange(e.target.value)}
											onBlur={field.handleBlur}
										/>
										{!field.state.meta.isValid ? (
											<FieldError errors={field.state.meta.errors} />
										) : null}
									</Field>
								)}
							</form.Field>
							<form.Field name="password">
								{(field) => (
									<Field id="password">
										<div className="flex items-center">
											<FieldLabel htmlFor="password">Password</FieldLabel>
											<span className="ml-auto text-sm text-muted-foreground">
												Reset unavailable
											</span>
										</div>
										<Input
											id="password"
											type="password"
											autoComplete="current-password"
											value={field.state.value}
											onChange={(e) => field.handleChange(e.target.value)}
											onBlur={field.handleBlur}
										/>
										{!field.state.meta.isValid ? (
											<FieldError errors={field.state.meta.errors} />
										) : null}
									</Field>
								)}
							</form.Field>

							<Field>
								<Button type="submit" disabled={loginMutation.isPending}>
									{loginMutation.isPending ? "Signing in…" : "Sign in"}
								</Button>
								{providers.length > 0 ? (
									<>
										<div className="relative my-2">
											<div className="absolute inset-0 flex items-center">
												<span className="w-full border-t" />
											</div>
											<div className="relative flex justify-center text-xs uppercase">
												<span className="bg-card px-2 text-muted-foreground">
													Or continue with
												</span>
											</div>
										</div>
										{providers.map((p) => (
											<Button
												key={p.id}
												type="button"
												variant="outline"
												onClick={() => startSSO(p.id, search.redirect)}
											>
												{p.display_name}
											</Button>
										))}
									</>
								) : null}
								<FieldDescription className="text-center">
									Self-serve signup is not enabled yet.{" "}
									<Link
										to="/auth/signup"
										className="underline-offset-4 hover:underline"
									>
										Learn more
									</Link>
								</FieldDescription>
							</Field>
						</FieldGroup>
					</form>
				</CardContent>
			</Card>
			<FieldDescription className="px-6 text-center">
				By continuing you agree to our <Link to="/terms">Terms</Link> and{" "}
				<Link to="/privacy">Privacy Policy</Link>.
			</FieldDescription>
		</div>
	);
}
