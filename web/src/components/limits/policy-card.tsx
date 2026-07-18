import {
  Building2,
  FolderKanban,
  MoreHorizontal,
  User,
  Users,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";

export type Policy = {
  id: string;
  name: string;
  enabled: boolean;
  scope: "organization" | "team" | "project" | "user";
  priority: number;
  models: string[];
  requests: string;
  inputTokens: string;
  outputTokens: string;
};

type Props = {
  policy: Policy;
  onClick(): void;
};

export function PolicyCard({
  policy,
  onClick,
}: Props) {
  const Icon = {
    organization: Building2,
    team: Users,
    project: FolderKanban,
    user: User,
  }[policy.scope];

  return (
    <Card
      className="cursor-pointer transition-colors hover:border-primary"
      onClick={onClick}
    >
      <CardContent className="flex items-center justify-between p-6">
        <div className="flex gap-4">
          <div className="rounded-md border p-3">
            <Icon className="h-5 w-5" />
          </div>

          <div>
            <div className="flex items-center gap-2">
              <h3 className="font-medium">{policy.name}</h3>

              <Badge
                variant={
                  policy.enabled ? "default" : "secondary"
                }
              >
                {policy.enabled ? "Enabled" : "Disabled"}
              </Badge>

              <Badge variant="outline">
                {policy.scope}
              </Badge>
            </div>

            <div className="mt-2 flex flex-wrap gap-2">
              {policy.models.map((model) => (
                <Badge
                  key={model}
                  variant="secondary"
                >
                  {model}
                </Badge>
              ))}
            </div>

            <div className="mt-3 flex gap-6 text-sm text-muted-foreground">
              <span>{policy.requests}</span>
              <span>{policy.inputTokens}</span>
              <span>{policy.outputTokens}</span>
              <span>Priority {policy.priority}</span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Switch
            checked={policy.enabled}
            onClick={(e) => e.stopPropagation()}
          />

          <Button
            variant="ghost"
            size="icon"
            onClick={(e) => e.stopPropagation()}
          >
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}