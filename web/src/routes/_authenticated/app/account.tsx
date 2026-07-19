import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { InfoIcon, LogOutIcon } from "lucide-react";
import { meQueryOptions } from "#/api/auth";
import { PageBody, PageHeader } from "#/components/page-header";
import { QueryGate } from "#/components/query-state";
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { Avatar, AvatarFallback } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import { useAuthActions } from "#/state/auth-state";

export const Route = createFileRoute("/_authenticated/app/account")({
	staticData: {
		getTitle: () => "Account",
	},
	component: RouteComponent,
});

function initials(name?: string, email?: string) {
	const source = name?.trim() || email || "?";
	return source
		.split(/\s+/)
		.slice(0, 2)
		.map((part) => part[0]?.toUpperCase() ?? "")
		.join("");
}

function RouteComponent() {
	const navigate = useNavigate();
	const { logout } = useAuthActions();
	const meQuery = useQuery(meQueryOptions());

	return (
		<PageBody className="max-w-2xl">
			<PageHeader
				title="Account"
				description="Your identity on the AFI control plane."
				actions={
					<Button
						variant="outline"
						onClick={() => {
							logout();
							navigate({ to: "/auth/login" });
						}}
					>
						<LogOutIcon />
						Log out
					</Button>
				}
			/>

			<QueryGate
				isPending={meQuery.isPending}
				isError={meQuery.isError}
				error={meQuery.error}
				onRetry={() => void meQuery.refetch()}
			>
				<Alert>
					<InfoIcon />
					<AlertTitle>Read-only profile</AlertTitle>
					<AlertDescription>
						Profile updates are not available in this build. Contact an admin to
						change identity details.
					</AlertDescription>
				</Alert>

				<Card>
					<CardHeader className="flex flex-row items-center gap-4">
						<Avatar className="size-14 rounded-lg">
							<AvatarFallback className="rounded-lg">
								{initials(meQuery.data?.name, meQuery.data?.email)}
							</AvatarFallback>
						</Avatar>
						<div className="space-y-1">
							<CardTitle>{meQuery.data?.name || "User"}</CardTitle>
							<CardDescription>{meQuery.data?.email}</CardDescription>
							{meQuery.data?.role ? (
								<Badge variant="secondary">{meQuery.data.role}</Badge>
							) : null}
						</div>
					</CardHeader>
					<CardContent>
						<FieldGroup>
							<Field>
								<FieldLabel>Full name</FieldLabel>
								<Input readOnly value={meQuery.data?.name ?? ""} />
							</Field>
							<Field>
								<FieldLabel>Email</FieldLabel>
								<Input readOnly value={meQuery.data?.email ?? ""} />
							</Field>
							<Field>
								<FieldLabel>User ID</FieldLabel>
								<Input
									readOnly
									value={meQuery.data?.id ?? ""}
									className="font-mono text-xs"
								/>
							</Field>
						</FieldGroup>
					</CardContent>
				</Card>
			</QueryGate>
		</PageBody>
	);
}
