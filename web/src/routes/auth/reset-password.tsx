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

export const Route = createFileRoute("/auth/reset-password")({
	...pageTitle("Reset password", {
		description: "Reset the password for your AFI account.",
	}),
	component: RouteComponent,
});

function RouteComponent() {
	return (
		<Card>
			<CardHeader className="text-center">
				<CardTitle>Password reset unavailable</CardTitle>
				<CardDescription>
					Password recovery is not available in this build. Contact your
					administrator if you need credentials rotated.
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
