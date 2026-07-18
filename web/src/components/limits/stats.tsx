import {
  Building2,
  FolderKanban,
  ShieldCheck,
  Users,
} from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const stats = [
  {
    title: "Policies",
    value: 12,
    description: "Active limit policies",
    icon: ShieldCheck,
  },
  {
    title: "Organizations",
    value: 1,
    description: "Protected",
    icon: Building2,
  },
  {
    title: "Teams",
    value: 8,
    description: "With custom policies",
    icon: Users,
  },
  {
    title: "Projects",
    value: 24,
    description: "Using inherited limits",
    icon: FolderKanban,
  },
];

export function LimitsStats() {
  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      {stats.map((stat) => {
        const Icon = stat.icon;

        return (
          <Card key={stat.title}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
              <CardTitle className="text-sm font-medium">
                {stat.title}
              </CardTitle>

              <Icon className="size-4 text-muted-foreground" />
            </CardHeader>

            <CardContent>
              <div className="text-3xl font-semibold">
                {stat.value}
              </div>

              <p className="mt-1 text-sm text-muted-foreground">
                {stat.description}
              </p>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}