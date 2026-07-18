import { createFileRoute } from "@tanstack/react-router";
import { Badge } from "@/components/ui/badge";
import {
  Plus,
  Search,
  Building2,
  Users,
  FolderKanban,
  User,
  ArrowUpRight,
  MoreHorizontal,
} from "lucide-react";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import { Input } from "#/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "#/components/ui/select";
import { Switch } from "#/components/ui/switch";

export const Route = createFileRoute("/_authenticated/app/settings/limits")({
  staticData: {
    getTitle: () => "Limits",
  },
  component: RouteComponent,
});

type Scope = "organization" | "team" | "project" | "user";

type Policy = {
  id: string;
  name: string;
  scope: Scope;
  enabled: boolean;

  models: string[];

  requests: string;
  inputTokens: string;
  outputTokens: string;

  priority: number;
};

const policies: Policy[] = [
  {
    id: "1",
    name: "Organization Default",
    scope: "organization",
    enabled: true,
    models: ["GPT-4o", "Claude Sonnet 4"],
    requests: "60 / min",
    inputTokens: "2M / day",
    outputTokens: "1M / day",
    priority: 100,
  },
  {
    id: "2",
    name: "Marketing Team",
    scope: "team",
    enabled: true,
    models: ["GPT-4.1"],
    requests: "120 / min",
    inputTokens: "5M / day",
    outputTokens: "2M / day",
    priority: 80,
  },
  {
    id: "3",
    name: "Project Alpha",
    scope: "project",
    enabled: false,
    models: ["Gemini 2.5"],
    requests: "500 / day",
    inputTokens: "-",
    outputTokens: "-",
    priority: 40,
  },
];

function RouteComponent() {
  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-8 p-8">
      {/* Header */}

      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Limits</h1>

          <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
            Control request volume and token usage across organizations, teams,
            projects and users.
          </p>
        </div>

        <Button>
          <Plus className="mr-2 h-4 w-4" />
          New Policy
        </Button>
      </div>

      {/* Stats */}

      <div className="grid gap-4 md:grid-cols-4">
        <StatCard title="Policies" value="12" description="Total configured" />

        <StatCard title="Enabled" value="10" description="Currently active" />

        <StatCard title="Scopes" value="4" description="Organization → User" />

        <StatCard
          title="Protected Models"
          value="18"
          description="Across all providers"
        />
      </div>

      {/* Toolbar */}

      <Card>
        <CardContent className="flex flex-col gap-4 p-4 md:flex-row">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />

            <Input placeholder="Search policies..." className="pl-9" />
          </div>

          <Select defaultValue="all">
            <SelectTrigger className="w-full md:w-48">
              <SelectValue />
            </SelectTrigger>

            <SelectContent>
              <SelectItem value="all">All Scopes</SelectItem>
              <SelectItem value="organization">Organization</SelectItem>
              <SelectItem value="team">Team</SelectItem>
              <SelectItem value="project">Project</SelectItem>
              <SelectItem value="user">User</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      {/* Policy List */}

      <div className="space-y-4">
        {policies.map((policy) => (
          <PolicyCard key={policy.id} policy={policy} />
        ))}
      </div>
    </div>
  );
}

function StatCard(props: {
  title: string;
  value: string;
  description: string;
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardDescription>{props.title}</CardDescription>

        <CardTitle className="text-3xl">{props.value}</CardTitle>
      </CardHeader>

      <CardContent className="text-sm text-muted-foreground">
        {props.description}
      </CardContent>
    </Card>
  );
}

function PolicyCard({ policy }: { policy: Policy }) {
  const ScopeIcon = {
    organization: Building2,
    team: Users,
    project: FolderKanban,
    user: User,
  }[policy.scope];

  return (
    <Card className="transition-colors hover:border-primary">
      <CardContent className="flex flex-col gap-6 p-6 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex gap-4">
          <div>
            <div className="rounded-lg border p-3">
              <ScopeIcon className="size-5" />
            </div>
          </div>

          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="font-medium">{policy.name}</h3>

              <Badge variant={policy.enabled ? "default" : "secondary"}>
                {policy.enabled ? "Enabled" : "Disabled"}
              </Badge>

              <Badge variant="outline">{policy.scope}</Badge>
            </div>

            <div className="mt-3 flex flex-wrap gap-2">
              {policy.models.map((model) => (
                <Badge key={model} variant="secondary">
                  {model}
                </Badge>
              ))}
            </div>

            <div className="mt-4 flex flex-wrap gap-6 text-sm text-muted-foreground">
              <span>
                <strong>Requests:</strong> {policy.requests}
              </span>

              <span>
                <strong>Input:</strong> {policy.inputTokens}
              </span>

              <span>
                <strong>Output:</strong> {policy.outputTokens}
              </span>

              <span>
                <strong>Priority:</strong> {policy.priority}
              </span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Switch checked={policy.enabled} />

          <Button variant="ghost" size="icon">
            <ArrowUpRight className="h-4 w-4" />
          </Button>

          <Button variant="ghost" size="icon">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
