import { createFileRoute, Link } from "@tanstack/react-router";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";
import { pageTitle } from "#/lib/page-meta";

export const Route = createFileRoute("/auth/signup")({
	...pageTitle("Sign up", {
		description:
			"Create an AFI account to manage organizations, keys, and gateway policies.",
	}),
	component: RouteComponent,
});

function RouteComponent() {
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
