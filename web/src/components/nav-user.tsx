"use client";

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
import {
  ChevronsUpDownIcon,
  SparklesIcon,
  BadgeCheckIcon,
  CreditCardIcon,
  BellIcon,
  LogOutIcon,
} from "lucide-react";
import { useAuthStore } from "#/state/auth-state";
import { useRouter } from "@tanstack/react-router";

type Organization = {
  id: string;
  name: string;
  logo: string;
};

type NavUserProps = {
  user: {
    name?: string;
    email: string;
    avatar?: string;
  };
  activeOrganization: Organization;
  organizations: Organization[];
  onOrganizationChange: (id: string) => void;
};

export function NavUser({
  user,
  activeOrganization,
  organizations,
  onOrganizationChange,
}: NavUserProps) {
  const { isMobile } = useSidebar();
  const router = useRouter();
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
              <AvatarImage
                src={activeOrganization.logo}
                alt={activeOrganization.name}
              />
              <AvatarFallback>
                {user.name || user.email.slice(0, 2).toUpperCase()}
              </AvatarFallback>
            </Avatar>

            <div className="grid flex-1 text-left leading-tight">
              <span className="truncate text-sm font-medium">
                {user.name || user.email}
              </span>

              <span className="truncate text-xs text-muted-foreground">
                {activeOrganization.name}
              </span>
            </div>

            <ChevronsUpDownIcon className="ml-auto size-4" />
          </DropdownMenuTrigger>

          <DropdownMenuContent
            className="w-fit"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel>Organization</DropdownMenuLabel>

              {organizations.map((org) => (
                <DropdownMenuItem
                  key={org.id}
                  onClick={() => onOrganizationChange(org.id)}
                >
                  <Avatar className="size-5 rounded-md">
                    <AvatarImage src={org.logo} />
                    <AvatarFallback>{org.name.slice(0, 2)}</AvatarFallback>
                  </Avatar>

                  <span className="flex-1 ml-2">{org.name}</span>

                  {org.id === activeOrganization.id && (
                    <BadgeCheckIcon className="size-4" />
                  )}
                </DropdownMenuItem>
              ))}

              <DropdownMenuSeparator />

              <DropdownMenuItem>+ Create organization</DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />

            <DropdownMenuGroup>
              <DropdownMenuLabel className="p-0 font-normal">
                <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                  <Avatar>
                    <AvatarImage src={user.avatar} alt={user.name} />
                    <AvatarFallback>CN</AvatarFallback>
                  </Avatar>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{user.name}</span>
                    <span className="truncate text-xs">{user.email}</span>
                  </div>
                </div>
              </DropdownMenuLabel>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem>
                <SparklesIcon />
                Upgrade to Pro
              </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem>
                <BadgeCheckIcon />
                Account
              </DropdownMenuItem>
              <DropdownMenuItem>
                <CreditCardIcon />
                Billing
              </DropdownMenuItem>
              <DropdownMenuItem>
                <BellIcon />
                Notifications
              </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => {
              useAuthStore.getState().actions.logout();
              router.navigate({ to: "/auth/login" });
            }}>
              <LogOutIcon />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
