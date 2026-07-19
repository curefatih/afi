"use client";

import { Link, useLocation } from "@tanstack/react-router";
import { ChevronRightIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Badge } from "@/components/ui/badge";
import {
	Collapsible,
	CollapsibleContent,
	CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
	SidebarGroup,
	SidebarGroupLabel,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarMenuSub,
	SidebarMenuSubButton,
	SidebarMenuSubItem,
} from "@/components/ui/sidebar";

export type NavItem = {
	title: string;
	url: string;
	icon?: ReactNode;
	badge?: string;
	items?: {
		title: string;
		url: string;
	}[];
};

export function NavMain({ label, items }: { label: string; items: NavItem[] }) {
	const { pathname } = useLocation();

	return (
		<SidebarGroup>
			<SidebarGroupLabel>{label}</SidebarGroupLabel>
			<SidebarMenu>
				{items.map((item) =>
					item.items && item.items.length > 0 ? (
						<Collapsible
							key={item.title}
							defaultOpen={
								pathname.startsWith(item.url) ||
								item.items.some((sub) => pathname.startsWith(sub.url))
							}
							className="group/collapsible"
							render={<SidebarMenuItem />}
						>
							<CollapsibleTrigger
								render={
									<SidebarMenuButton
										tooltip={item.title}
										className={
											pathname.startsWith(item.url) ||
											item.items.some((sub) => pathname.startsWith(sub.url))
												? "bg-sidebar-accent text-sidebar-accent-foreground"
												: ""
										}
									/>
								}
							>
								{item.icon}
								<span>{item.title}</span>
								{item.badge ? (
									<Badge variant="outline" className="ml-auto">
										{item.badge}
									</Badge>
								) : (
									<ChevronRightIcon className="ml-auto transition-transform duration-200 group-data-open/collapsible:rotate-90" />
								)}
							</CollapsibleTrigger>
							<CollapsibleContent>
								<SidebarMenuSub>
									{item.items.map((subItem) => {
										const isCurrentSubActive = pathname === subItem.url;
										return (
											<SidebarMenuSubItem key={subItem.title}>
												<SidebarMenuSubButton
													render={
														<Link
															to={subItem.url}
															className={
																isCurrentSubActive
																	? "bg-sidebar-accent text-sidebar-accent-foreground font-semibold"
																	: "text-muted-foreground hover:text-foreground"
															}
														/>
													}
												>
													<span>{subItem.title}</span>
												</SidebarMenuSubButton>
											</SidebarMenuSubItem>
										);
									})}
								</SidebarMenuSub>
							</CollapsibleContent>
						</Collapsible>
					) : (
						<SidebarMenuItem key={item.title}>
							<SidebarMenuButton
								tooltip={item.title}
								render={
									<Link
										to={item.url}
										className={
											pathname === item.url ||
											(item.url !== "/app/dashboard" &&
												pathname.startsWith(item.url))
												? "bg-sidebar-accent text-sidebar-accent-foreground"
												: ""
										}
									/>
								}
							>
								{item.icon}
								<span>{item.title}</span>
								{item.badge ? (
									<Badge variant="outline" className="ml-auto">
										{item.badge}
									</Badge>
								) : null}
							</SidebarMenuButton>
						</SidebarMenuItem>
					),
				)}
			</SidebarMenu>
		</SidebarGroup>
	);
}
