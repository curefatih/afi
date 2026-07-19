import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";
import {
	acceptInviteMutationOptions,
	invitePreviewQueryOptions,
} from "#/api/organization";
import { QueryGate } from "#/components/query-state";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { pageTitle } from "#/lib/page-meta";
import { useAuthActions } from "#/state/auth-state";

export const Route = createFileRoute("/auth/invite/$token")({
	...pageTitle("Accept invite", {
		description: "Accept an organization invite and join AFI.",
	}),
	component: RouteComponent,
});

function RouteComponent() {
	const { token } = Route.useParams();
	const navigate = useNavigate();
	const { setUser } = useAuthActions();
	const [name, setName] = useState("");
	const [password, setPassword] = useState("");

	const preview = useQuery(invitePreviewQueryOptions(token));
	const accept = useMutation(acceptInviteMutationOptions());

	function onAccepted(data: {
		user: { id: string; email: string; name: string; role: string };
		token: string;
	}) {
		setUser({
			id: data.user.id,
			email: data.user.email,
			name: data.user.name,
			role: data.user.role,
			accessToken: data.token,
			refreshToken: null,
		});
		toast.success("Welcome to the organization");
		void navigate({ to: "/app/dashboard" });
	}

	return (
		<div className="flex flex-col gap-6">
			<Card>
				<CardHeader className="text-center">
					<CardTitle className="text-xl">Accept invite</CardTitle>
					<CardDescription>
						Join an organization on AFI with your invite link.
					</CardDescription>
				</CardHeader>
				<CardContent>
					<QueryGate
						isPending={preview.isPending}
						isError={preview.isError}
						error={preview.error}
						onRetry={() => void preview.refetch()}
					>
						{preview.data ? (
							<form
								className="flex flex-col gap-4"
								onSubmit={(e) => {
									e.preventDefault();
									accept.mutate(
										{
											token,
											name: preview.data.user_exists ? undefined : name,
											password: preview.data.user_exists ? undefined : password,
										},
										{
											onSuccess: onAccepted,
											onError: (err) =>
												toast.error(err.message || "Could not accept invite"),
										},
									);
								}}
							>
								<div className="space-y-1 text-sm">
									<p>
										<span className="text-muted-foreground">Organization:</span>{" "}
										{preview.data.organization_name}
									</p>
									<p>
										<span className="text-muted-foreground">Email:</span>{" "}
										{preview.data.email}
									</p>
								</div>

								{preview.data.user_exists ? (
									<p className="text-muted-foreground text-sm">
										An account already exists for this email. Accept to join the
										organization, then continue signed in.
									</p>
								) : (
									<>
										<div className="space-y-1">
											<Label htmlFor="invite-name">Name</Label>
											<Input
												id="invite-name"
												value={name}
												onChange={(e) => setName(e.target.value)}
												required
											/>
										</div>
										<div className="space-y-1">
											<Label htmlFor="invite-password">Password</Label>
											<Input
												id="invite-password"
												type="password"
												value={password}
												onChange={(e) => setPassword(e.target.value)}
												minLength={8}
												required
											/>
										</div>
									</>
								)}

								<Button type="submit" disabled={accept.isPending}>
									{accept.isPending ? "Joining…" : "Accept invite"}
								</Button>
								<Button
									type="button"
									variant="outline"
									nativeButton={false}
									render={<Link to="/auth/login" />}
								>
									Back to sign in
								</Button>
							</form>
						) : null}
					</QueryGate>
				</CardContent>
			</Card>
		</div>
	);
}
