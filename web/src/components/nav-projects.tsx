"use client";

import { Link, useLocation } from "@tanstack/react-router";
import { FolderKanbanIcon, MoreHorizontalIcon } from "lucide-react";
import {
	SidebarGroup,
	SidebarGroupLabel,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
} from "@/components/ui/sidebar";

export function NavProjects({
	projects,
}: {
	projects: { id: string; name: string }[];
}) {
	const { pathname } = useLocation();
	const visible = projects.slice(0, 5);

	return (
		<SidebarGroup className="group-data-[collapsible=icon]:hidden">
			<SidebarGroupLabel>Projects</SidebarGroupLabel>
			<SidebarMenu>
				{visible.map((project) => {
					const url = `/app/projects/${project.id}`;
					return (
						<SidebarMenuItem key={project.id}>
							<SidebarMenuButton
								render={
									<Link
										to="/app/projects/$projectId"
										params={{ projectId: project.id }}
										className={
											pathname === url
												? "bg-sidebar-accent text-sidebar-accent-foreground"
												: ""
										}
									/>
								}
							>
								<FolderKanbanIcon />
								<span>{project.name}</span>
							</SidebarMenuButton>
						</SidebarMenuItem>
					);
				})}
				<SidebarMenuItem>
					<SidebarMenuButton
						className="text-sidebar-foreground/70"
						render={<Link to="/app/projects" />}
					>
						<MoreHorizontalIcon className="text-sidebar-foreground/70" />
						<span>All projects</span>
					</SidebarMenuButton>
				</SidebarMenuItem>
			</SidebarMenu>
		</SidebarGroup>
	);
}
