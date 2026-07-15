"use client";

import * as React from "react";

import { organizationsQueryOptions } from "#/api/user";
import { ProjectSwitcher } from "#/components/project-switcher";
import { useAuthUser } from "#/state/auth-state";
import { useOrgStore } from "#/state/organization-state";
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
import { useMutation } from "@tanstack/react-query";
import {
  AudioLinesIcon,
  FishIcon,
  FrameIcon,
  GalleryVerticalEndIcon,
  Settings2Icon,
  TerminalIcon,
  TerminalSquareIcon,
  Users2,
} from "lucide-react";
import { useEffect } from "react";

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
      url: "/app/playground/chat",
      icon: <TerminalSquareIcon />,
      items: [
        {
          title: "Chat",
          url: "/app/playground/chat",
        },
        {
          title: "TTS",
          url: "/app/playground/tts",
        },
        {
          title: "STT",
          url: "/app/playground/stt",
        },
      ],
    },
    {
      title: "Teams",
      url: "/app/teams",
      icon: <Users2 />,
    },
    {
      title: "Hooks",
      url: "/app/hooks",
      icon: <FishIcon />,
    },
    {
      title: "Settings",
      url: "/app/settings/general",
      icon: <Settings2Icon />,
      items: [
        {
          title: "General",
          url: "/app/settings/general",
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
  const organizations = useOrgStore();
  if (!user) {
    return null;
  }

  const organizationsMutation = useMutation({
    ...organizationsQueryOptions(),
  });

  useEffect(() => {
    organizationsMutation.mutate(undefined);
  }, []);

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
          activeOrganization={organizations.activeOrg!}
          organizations={organizations.orgs}
          onOrganizationChange={() => {}}
        />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
