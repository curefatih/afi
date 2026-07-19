"use client";

import { useNavigate } from "@tanstack/react-router";
import { BadgeCheckIcon, ChevronsUpDownIcon, LogOutIcon } from "lucide-react";
import { useAuthStore } from "#/state/auth-state";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
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

function initials(name?: string, email?: string) {
	const source = name?.trim() || email || "?";
	return source
		.split(/\s+/)
		.slice(0, 2)
		.map((part) => part[0]?.toUpperCase() ?? "")
		.join("");
}

type NavUserProps = {
	user: {
		name?: string;
		email: string;
		avatar?: string;
	};
};

export function NavUser({ user }: NavUserProps) {
	const { isMobile } = useSidebar();
	const navigate = useNavigate();

	return (
		<SidebarMenu>
			<SidebarMenuItem>
				<DropdownMenu>
					<DropdownMenuTrigger
						render={
							<SidebarMenuButton size="lg" className="aria-expanded:bg-muted" />
						}
					>
						<Avatar className="rounded-md">
							<AvatarImage src={user.avatar} alt={user.name || user.email} />
							<AvatarFallback>{initials(user.name, user.email)}</AvatarFallback>
						</Avatar>

						<div className="grid flex-1 text-left leading-tight">
							<span className="truncate text-sm font-medium">
								{user.name || user.email}
							</span>
							<span className="truncate text-xs text-muted-foreground">
								{user.email}
							</span>
						</div>

						<ChevronsUpDownIcon className="ml-auto size-4" />
					</DropdownMenuTrigger>

					<DropdownMenuContent
						className="w-56"
						side={isMobile ? "bottom" : "right"}
						align="end"
						sideOffset={4}
					>
						<DropdownMenuGroup>
							<DropdownMenuLabel className="p-0 font-normal">
								<div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
									<Avatar>
										<AvatarImage src={user.avatar} alt={user.name} />
										<AvatarFallback>
											{initials(user.name, user.email)}
										</AvatarFallback>
									</Avatar>
									<div className="grid flex-1 text-left text-sm leading-tight">
										<span className="truncate font-medium">
											{user.name || "Account"}
										</span>
										<span className="truncate text-xs text-muted-foreground">
											{user.email}
										</span>
									</div>
								</div>
							</DropdownMenuLabel>
						</DropdownMenuGroup>
						<DropdownMenuSeparator />
						<DropdownMenuGroup>
							<DropdownMenuItem
								onClick={() => {
									navigate({ to: "/app/account" });
								}}
							>
								<BadgeCheckIcon />
								Account
							</DropdownMenuItem>
						</DropdownMenuGroup>
						<DropdownMenuSeparator />
						<DropdownMenuItem
							onClick={() => {
								useAuthStore.getState().actions.logout();
								navigate({ to: "/auth/login" });
							}}
						>
							<LogOutIcon />
							Log out
						</DropdownMenuItem>
					</DropdownMenuContent>
				</DropdownMenu>
			</SidebarMenuItem>
		</SidebarMenu>
	);
}
