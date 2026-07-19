"use client";

import * as React from "react";

import { organizationsQueryOptions } from "#/api/user";
import { TeamSwitcher } from "#/components/team-switcher";
import { useAuthUser } from "#/state/auth-state";
import {
  useActiveOrg,
  useActiveTeam,
  useOrgStore,
} from "#/state/organization-state";
import { NavMain } from "@/components/nav-main";
import { NavTeam } from "#/components/nav-team";
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
import { Empty, EmptyContent, EmptyDescription } from "./ui/empty";
import { useNavigate } from "@tanstack/react-router";

// This is sample data.
const data = {
  organizationSwitcher: [
    {
      name: "AFI Inc",
      team: "Enterprise",
    },
    {
      name: "Acme Corp.",
      team: "Startup",
    },
    {
      name: "Evil Corp.",
      team: "Free",
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
};

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const navigate = useNavigate();
  const user = useAuthUser();
  const organizations = useOrgStore();
  const activeOrg = useActiveOrg();
  const activeTeam = useActiveTeam();

  const organizationsMutation = useMutation({
    ...organizationsQueryOptions(),
  });

  useEffect(() => {
    organizationsMutation.mutate(undefined);
  }, []);

  if (!user) {
    navigate({ to: "/auth/login" });
  }

  if (!user || !activeOrg) {
    return null;
  }
  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        {!activeOrg ? (
          "You dont have any organization yet"
        ) : (
          <TeamSwitcher teams={activeOrg.teams} />
        )}
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
        {!activeOrg || !activeTeam ? (
          <Empty>
            <EmptyContent>
              <EmptyDescription>Select team for team options</EmptyDescription>
            </EmptyContent>
          </Empty>
        ) : (
          <>
            <NavTeam
              projects={activeOrg.projects.map((p) => ({
                name: p.name,
                url: `/app/projects/${p.id}`,
              }))}
            />
          </>
        )}
      </SidebarContent>
      <SidebarFooter>
        <NavUser
          user={user}
          activeOrganization={activeOrg!}
          organizations={organizations.orgs}
          onOrganizationChange={() => {}}
        />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
