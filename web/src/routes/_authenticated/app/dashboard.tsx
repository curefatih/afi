import { createFileRoute, Link } from "@tanstack/react-router";
import {
	FolderKanbanIcon,
	KeyRoundIcon,
	TerminalSquareIcon,
	Users2Icon,
} from "lucide-react";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";
import { useOrgBootstrap } from "#/hooks/use-org-bootstrap";
import { useActiveOrg } from "#/state/organization-state";

export const Route = createFileRoute("/_authenticated/app/dashboard")({
	staticData: {
		getTitle: () => "Overview",
	},
	component: RouteComponent,
});

function RouteComponent() {
	const activeOrg = useActiveOrg();
	const { isBootstrapping, isError, error, refetch } = useOrgBootstrap();

	const teamCount = activeOrg?.teams.length ?? 0;
	const projectCount = activeOrg?.projects.length ?? 0;

	return (
		<PageBody>
			<PageHeader
				title={activeOrg ? activeOrg.name : "Overview"}
				description="Control-plane summary for the active organization. Configure projects and keys, then exercise traffic in the playground."
			/>

			<QueryGate
				isPending={isBootstrapping && !activeOrg}
				isError={isError}
				error={error}
				onRetry={refetch}
			>
				<div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
					<Card>
						<CardHeader>
							<CardDescription>Teams</CardDescription>
							<CardTitle className="text-3xl tabular-nums">
								{teamCount}
							</CardTitle>
						</CardHeader>
						<CardContent>
							<Button
								variant="outline"
								size="sm"
								render={<Link to="/app/teams" />}
							>
								<Users2Icon />
								View teams
							</Button>
						</CardContent>
					</Card>
					<Card>
						<CardHeader>
							<CardDescription>Projects</CardDescription>
							<CardTitle className="text-3xl tabular-nums">
								{projectCount}
							</CardTitle>
						</CardHeader>
						<CardContent>
							<Button
								variant="outline"
								size="sm"
								render={<Link to="/app/projects" />}
							>
								<FolderKanbanIcon />
								View projects
							</Button>
						</CardContent>
					</Card>
					<Card>
						<CardHeader>
							<CardDescription>API Keys</CardDescription>
							<CardTitle className="text-base">Virtual keys</CardTitle>
						</CardHeader>
						<CardContent>
							<Button
								variant="outline"
								size="sm"
								render={<Link to="/app/keys" />}
							>
								<KeyRoundIcon />
								Manage keys
							</Button>
						</CardContent>
					</Card>
					<Card>
						<CardHeader>
							<CardDescription>Playground</CardDescription>
							<CardTitle className="text-base">Chat</CardTitle>
						</CardHeader>
						<CardContent>
							<Button
								variant="outline"
								size="sm"
								render={<Link to="/app/playground/chat" />}
							>
								<TerminalSquareIcon />
								Open playground
							</Button>
						</CardContent>
					</Card>
				</div>

				<Card>
					<CardHeader>
						<CardTitle>Local development</CardTitle>
						<CardDescription>
							Seeded credentials for local control plane and gateway.
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-2 text-sm text-muted-foreground">
						<p>
							Login:{" "}
							<span className="font-mono text-foreground">admin@afi.local</span>{" "}
							/ <span className="font-mono text-foreground">admin</span>
						</p>
						<p>
							Ensure control plane (:8081) and gateway (:8080) are running, then
							publish a snapshot after seed if needed.
						</p>
					</CardContent>
				</Card>
			</QueryGate>
		</PageBody>
	);
}
