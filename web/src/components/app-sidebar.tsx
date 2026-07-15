"use client";

import * as React from "react";

import { ProjectSwitcher } from "#/components/project-switcher";
import { useAuthUser } from "#/state/auth-state";
import { NavMain } from "@/components/nav-main";
import { NavProjects } from "@/components/nav-projects";
import { NavUser } from "@/components/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
import {
  AudioLinesIcon,
  FrameIcon,
  GalleryVerticalEndIcon,
  MapIcon,
  PieChartIcon,
  Settings2Icon,
  TerminalIcon,
  TerminalSquareIcon,
  Users2
} from "lucide-react";

// This is sample data.
const data = {
  organizationSwitcher: [
    {
      name: "AFI Inc",
      logo: <GalleryVerticalEndIcon />,
      plan: "Enterprise",
    },
    {
      name: "Acme Corp.",
      logo: <AudioLinesIcon />,
      plan: "Startup",
    },
    {
      name: "Evil Corp.",
      logo: <TerminalIcon />,
      plan: "Free",
    },
  ],
  navMain: [
    {
      title: "Playground",
      url: "/app/playground",
      icon: <TerminalSquareIcon />,
      // isActive: true,
      // items: [
      //   {
      //     title: "History",
      //     url: "#",
      //   },
      //   {
      //     title: "Starred",
      //     url: "#",
      //   },
      //   {
      //     title: "Settings",
      //     url: "#",
      //   },
      // ],
    },
    {
      title: "Teams",
      url: "/app/teams",
      icon: <Users2 />,
      // isActive: true,
      // items: [
      //   {
      //     title: "History",
      //     url: "#",
      //   },
      //   {
      //     title: "Starred",
      //     url: "#",
      //   },
      //   {
      //     title: "Settings",
      //     url: "#",
      //   },
      // ],
    },
    {
      title: "Settings",
      url: "/app/settings",
      icon: <Settings2Icon />,
      items: [
        {
          title: "General",
          url: "/app/settings",
        },
        {
          title: "Team",
          url: "/app/settings/teams",
        },
        {
          title: "Limits",
          url: "/app/settings/limits",
        },
      ],
    },
  ],
  projects: [
    {
      name: "Design Engineering",
      url: "#",
      icon: <FrameIcon />,
    },
    {
      name: "Sales & Marketing",
      url: "#",
      icon: <PieChartIcon />,
    },
    {
      name: "Travel",
      url: "#",
      icon: <MapIcon />,
    },
  ],

  organizations: [
    {
      id: "org_1",
      name: "AFI Inc.",
      logo: "https://github.com/vercel.png",
    },
    {
      id: "org_2",
      name: "Personal",
      logo: "https://github.com/shadcn.png",
    },
    {
      id: "org_3",
      name: "AI Research",
      logo: "https://github.com/openai.png",
    },
  ],
};

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const user = useAuthUser();
  if (!user) {
    return null;
  }

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <ProjectSwitcher projects={data.organizationSwitcher} />
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
        <NavProjects projects={data.projects} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser
          user={user}
          activeOrganization={data.organizations[0]}
          organizations={data.organizations}
          onOrganizationChange={() => {}}
        />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
