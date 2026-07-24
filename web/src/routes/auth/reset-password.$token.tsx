import { useForm } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { z } from "zod";
import { confirmPasswordResetMutationOptions } from "#/api/auth";
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

const resetFormSchema = z.object({
	password: z.string().min(8, "Password must be at least 8 characters"),
});

export const Route = createFileRoute("/auth/reset-password/$token")({
	...pageTitle("Reset password", {
		description: "Choose a new password for your AFI account.",
	}),
	component: RouteComponent,
});

function RouteComponent() {
	const { token } = Route.useParams();
	const navigate = useNavigate();
	const confirmMutation = useMutation(confirmPasswordResetMutationOptions());

	const form = useForm({
		validators: {
			onChange: resetFormSchema,
		},
		defaultValues: {
			password: "",
		},
		onSubmit: async (values) => {
			confirmMutation.mutate(
				{ token, password: values.value.password },
				{
					onSuccess: () => {
						toast.success("Password updated");
						void navigate({ to: "/app/dashboard" });
					},
					onError: (error) => {
						toast.error(error.message || "Could not reset password");
					},
				},
			);
		},
	});

	return (
		<div className="flex flex-col gap-6">
			<Card>
				<CardHeader className="text-center">
					<CardTitle className="text-xl">Choose a new password</CardTitle>
					<CardDescription>
						Enter a new password for your account. You will be signed in after
						saving.
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
							<form.Field name="password">
								{(field) => (
									<Field id="password">
										<FieldLabel htmlFor="password">New password</FieldLabel>
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
								<Button type="submit" disabled={confirmMutation.isPending}>
									{confirmMutation.isPending ? "Updating…" : "Update password"}
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
