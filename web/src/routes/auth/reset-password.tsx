import { createFileRoute, Link } from "@tanstack/react-router";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";

export const Route = createFileRoute("/auth/reset-password")({
	staticData: {
		getTitle: () => "Reset password",
	},
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
				<Button render={<Link to="/auth/login" />}>Back to sign in</Button>
			</CardContent>
		</Card>
	);
}
