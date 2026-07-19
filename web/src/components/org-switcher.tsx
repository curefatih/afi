"use client";

import { useNavigate } from "@tanstack/react-router";
import {
	Building2Icon,
	CheckIcon,
	ChevronsUpDownIcon,
	Settings2Icon,
} from "lucide-react";
import {
	type Organization,
	useActiveOrg,
	useOrgActions,
} from "#/state/organization-state";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	useSidebar,
} from "@/components/ui/sidebar";

export function OrgSwitcher({
	organizations,
}: {
	organizations: Organization[];
}) {
	const navigate = useNavigate();
	const { isMobile } = useSidebar();
	const activeOrg = useActiveOrg();
	const { setActiveOrgById } = useOrgActions();

	if (!activeOrg) {
		return (
			<SidebarMenu>
				<SidebarMenuItem>
					<SidebarMenuButton size="lg" disabled>
						<div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
							<Building2Icon className="size-4" />
						</div>
						<div className="grid flex-1 text-left text-sm leading-tight">
							<span className="truncate font-medium">No organization</span>
							<span className="truncate text-xs text-muted-foreground">
								Join or create an org
							</span>
						</div>
					</SidebarMenuButton>
				</SidebarMenuItem>
			</SidebarMenu>
		);
	}

	return (
		<SidebarMenu>
			<SidebarMenuItem>
				<DropdownMenu>
					<DropdownMenuTrigger
						render={
							<SidebarMenuButton
								size="lg"
								className="data-open:bg-sidebar-accent data-open:text-sidebar-accent-foreground"
							/>
						}
					>
						<div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
							<Building2Icon className="size-4" />
						</div>
						<div className="grid flex-1 text-left text-sm leading-tight">
							<span className="truncate font-medium">{activeOrg.name}</span>
							<span className="truncate text-xs text-muted-foreground">
								Organization
							</span>
						</div>
						<ChevronsUpDownIcon className="ml-auto" />
					</DropdownMenuTrigger>
					<DropdownMenuContent
						className="w-64"
						align="start"
						side={isMobile ? "bottom" : "right"}
						sideOffset={4}
					>
						<DropdownMenuGroup>
							<DropdownMenuLabel className="text-xs text-muted-foreground">
								Organizations
							</DropdownMenuLabel>
							{organizations.map((org) => (
								<DropdownMenuItem
									key={org.id}
									onClick={() => setActiveOrgById(org.id)}
									className="gap-2 p-2"
								>
									<div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
										<Building2Icon className="size-3.5" />
									</div>
									<span className="flex-1 truncate">{org.name}</span>
									{org.id === activeOrg.id ? (
										<CheckIcon className="size-4" />
									) : null}
								</DropdownMenuItem>
							))}
						</DropdownMenuGroup>
						<DropdownMenuSeparator />
						<DropdownMenuGroup>
							<DropdownMenuItem
								className="gap-2 p-2"
								onClick={() => {
									void navigate({ to: "/app/settings/general" });
								}}
							>
								<div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
									<Settings2Icon className="size-3.5" />
								</div>
								<span className="flex-1 truncate">Organization settings</span>
							</DropdownMenuItem>
						</DropdownMenuGroup>
					</DropdownMenuContent>
				</DropdownMenu>
			</SidebarMenuItem>
		</SidebarMenu>
	);
}
