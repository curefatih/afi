"use client";

import {
	BarChart3Icon,
	Building2Icon,
	CreditCardIcon,
	FolderKanbanIcon,
	GaugeIcon,
	KeyRoundIcon,
	LayoutDashboardIcon,
	PlugIcon,
	PuzzleIcon,
	RouteIcon,
	ScrollTextIcon,
	Settings2Icon,
	ShieldIcon,
	TerminalSquareIcon,
	UserRoundIcon,
	Users2Icon,
} from "lucide-react";
import { NavMain } from "#/components/nav-main";
import { NavProjects } from "#/components/nav-projects";
import { NavUser } from "#/components/nav-user";
import { OrgSwitcher } from "#/components/org-switcher";
import { useOrgBootstrap } from "#/hooks/use-org-bootstrap";
import { useAuthUser } from "#/state/auth-state";
import { useActiveOrg, useOrgStore } from "#/state/organization-state";
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarHeader,
	SidebarRail,
} from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";

const platformNav = [
	{
		title: "Overview",
		url: "/app/dashboard",
		icon: <LayoutDashboardIcon />,
	},
	{
		title: "Organizations",
		url: "/app/organizations",
		icon: <Building2Icon />,
	},
	{
		title: "Settings",
		url: "/app/settings/general",
		icon: <Settings2Icon />,
	},
	{
		title: "Projects",
		url: "/app/projects",
		icon: <FolderKanbanIcon />,
	},
	{
		title: "Teams",
		url: "/app/teams",
		icon: <Users2Icon />,
	},
	{
		title: "Users",
		url: "/app/users",
		icon: <UserRoundIcon />,
	},
	{
		title: "API Keys",
		url: "/app/keys",
		icon: <KeyRoundIcon />,
	},
	{
		title: "Playground",
		url: "/app/playground/chat",
		icon: <TerminalSquareIcon />,
		items: [
			{ title: "Chat", url: "/app/playground/chat" },
			{ title: "TTS", url: "/app/playground/tts" },
			{ title: "STT", url: "/app/playground/stt" },
		],
	},
];

const governanceNav = [
	{
		title: "Providers",
		url: "/app/providers",
		icon: <PlugIcon />,
	},
	{
		title: "Routing",
		url: "/app/routing",
		icon: <RouteIcon />,
	},
	{
		title: "Quotas",
		url: "/app/quotas",
		icon: <GaugeIcon />,
	},
	{
		title: "Policies",
		url: "/app/policies",
		icon: <ScrollTextIcon />,
	},
	{
		title: "Usage",
		url: "/app/usage",
		icon: <BarChart3Icon />,
	},
	// {
	// 	title: "Billing",
	// 	url: "/app/billing",
	// 	icon: <CreditCardIcon />,
	// 	badge: "Soon",
	// },
	// {
	// 	title: "Secrets",
	// 	url: "/app/secrets",
	// 	icon: <ShieldIcon />,
	// 	badge: "Soon",
	// },
	{
		title: "Hooks",
		url: "/app/hooks",
		icon: <PuzzleIcon />,
	},
];

function SidebarSkeleton() {
	return (
		<div className="flex flex-col gap-4 p-4">
			<Skeleton className="h-12 w-full" />
			<Skeleton className="h-4 w-20" />
			<Skeleton className="h-8 w-full" />
			<Skeleton className="h-8 w-full" />
			<Skeleton className="h-8 w-full" />
			<Skeleton className="h-4 w-24 mt-4" />
			<Skeleton className="h-8 w-full" />
			<Skeleton className="h-8 w-full" />
		</div>
	);
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
	const user = useAuthUser();
	const orgs = useOrgStore((s) => s.orgs);
	const activeOrg = useActiveOrg();
	const { isBootstrapping } = useOrgBootstrap();

	return (
		<Sidebar collapsible="icon" {...props}>
			<SidebarHeader>
				{isBootstrapping && !activeOrg ? (
					<Skeleton className="h-12 w-full" />
				) : (
					<OrgSwitcher organizations={orgs} />
				)}
			</SidebarHeader>
			<SidebarContent>
				{isBootstrapping && !activeOrg ? (
					<SidebarSkeleton />
				) : (
					<>
						<NavMain label="Platform" items={platformNav} />
						{activeOrg && activeOrg.projects.length > 0 ? (
							<NavProjects projects={activeOrg.projects} />
						) : null}
						<NavMain label="Governance" items={governanceNav} />
					</>
				)}
			</SidebarContent>
			<SidebarFooter>
				{user ? <NavUser user={user} /> : <Skeleton className="h-12 w-full" />}
			</SidebarFooter>
			<SidebarRail />
		</Sidebar>
	);
}
