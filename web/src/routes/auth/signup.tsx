import { createFileRoute, Link } from "@tanstack/react-router";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";

export const Route = createFileRoute("/auth/signup")({
	staticData: {
		getTitle: () => "Sign up",
	},
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
