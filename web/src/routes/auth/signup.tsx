import { useForm } from "@tanstack/react-form";
import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { z } from "zod";
import { authFeaturesQueryOptions, registerMutationOptions } from "#/api/auth";
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
import { pageTitle } from "#/lib/page-meta";

const signupFormSchema = z.object({
	name: z.string().min(1, "Name is required"),
	email: z.email("Email is required"),
	password: z.string().min(8, "Password must be at least 8 characters"),
});

export const Route = createFileRoute("/auth/signup")({
	...pageTitle("Sign up", {
		description:
			"Create an AFI account to manage organizations, keys, and gateway policies.",
	}),
	component: RouteComponent,
});

function RouteComponent() {
	const navigate = useNavigate();
	const features = useQuery(authFeaturesQueryOptions());
	const registerMutation = useMutation(registerMutationOptions());

	const form = useForm({
		validators: {
			onChange: signupFormSchema,
		},
		defaultValues: {
			name: "",
			email: "",
			password: "",
		},
		onSubmit: async (values) => {
			registerMutation.mutate(
				{
					name: values.value.name,
					email: values.value.email,
					password: values.value.password,
				},
				{
					onSuccess: () => {
						toast.success("Account created");
						void navigate({ to: "/app/dashboard" });
					},
					onError: (error) => {
						toast.error(error.message || "Sign up failed");
					},
				},
			);
		},
	});

	if (features.isPending) {
		return (
			<Card>
				<CardHeader className="text-center">
					<CardTitle>Sign up</CardTitle>
					<CardDescription>Loading…</CardDescription>
				</CardHeader>
			</Card>
		);
	}

	if (!features.data?.signup_enabled) {
		return (
			<Card>
				<CardHeader className="text-center">
					<CardTitle>Sign up unavailable</CardTitle>
					<CardDescription>
						Self-serve registration is not enabled for this deployment. Ask an
						administrator to provision your account.
					</CardDescription>
				</CardHeader>
				<CardContent className="flex justify-center">
					<Button nativeButton={false} render={<Link to="/auth/login" />}>
						Back to sign in
					</Button>
				</CardContent>
			</Card>
		);
	}

	return (
		<div className="flex flex-col gap-6">
			<Card>
				<CardHeader className="text-center">
					<CardTitle className="text-xl">Create your account</CardTitle>
					<CardDescription>
						Sign up with email and password. You can create or join an
						organization after signing in.
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
							<form.Field name="name">
								{(field) => (
									<Field id="name">
										<FieldLabel htmlFor="name">Name</FieldLabel>
										<Input
											id="name"
											autoComplete="name"
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
							<form.Field name="email">
								{(field) => (
									<Field id="email">
										<FieldLabel htmlFor="email">Email</FieldLabel>
										<Input
											id="email"
											type="email"
											autoComplete="email"
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
										<FieldLabel htmlFor="password">Password</FieldLabel>
										<Input
											id="password"
											type="password"
											autoComplete="new-password"
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
								<Button type="submit" disabled={registerMutation.isPending}>
									{registerMutation.isPending
										? "Creating account…"
										: "Create account"}
								</Button>
								<FieldDescription className="text-center">
									Already have an account?{" "}
									<Link
										to="/auth/login"
										className="underline-offset-4 hover:underline"
									>
										Sign in
									</Link>
								</FieldDescription>
							</Field>
						</FieldGroup>
					</form>
				</CardContent>
			</Card>
		</div>
	);
}
