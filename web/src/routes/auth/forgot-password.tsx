import { useForm } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";
import { z } from "zod";
import { requestPasswordResetMutationOptions } from "#/api/auth";
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

const forgotFormSchema = z.object({
	email: z.email("Email is required"),
});

export const Route = createFileRoute("/auth/forgot-password")({
	...pageTitle("Forgot password", {
		description: "Request a password reset link for your AFI account.",
	}),
	component: RouteComponent,
});

function RouteComponent() {
	const [sent, setSent] = useState(false);
	const resetMutation = useMutation(requestPasswordResetMutationOptions());

	const form = useForm({
		validators: {
			onChange: forgotFormSchema,
		},
		defaultValues: {
			email: "",
		},
		onSubmit: async (values) => {
			resetMutation.mutate(values.value.email, {
				onSuccess: () => {
					setSent(true);
					toast.success("Check your inbox");
				},
				onError: (error) => {
					toast.error(error.message || "Could not send reset email");
				},
			});
		},
	});

	if (sent) {
		return (
			<Card>
				<CardHeader className="text-center">
					<CardTitle>Check your inbox</CardTitle>
					<CardDescription>
						If an account exists for that email, we sent a password reset link.
						The link expires in one hour.
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
					<CardTitle className="text-xl">Forgot password</CardTitle>
					<CardDescription>
						Enter your email and we will send a reset link if an account exists.
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
								<Button type="submit" disabled={resetMutation.isPending}>
									{resetMutation.isPending ? "Sending…" : "Send reset link"}
								</Button>
								<FieldDescription className="text-center">
									<Link
										to="/auth/login"
										className="underline-offset-4 hover:underline"
									>
										Back to sign in
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
