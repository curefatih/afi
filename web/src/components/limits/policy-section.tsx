import { Card, CardContent } from "../ui/card";
import { Input } from "../ui/input";
import { Label } from "../ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select";
import { Separator } from "../ui/separator";
import { Switch } from "../ui/switch";
import { usePolicyForm } from "./policy-form-provider";

export function PolicySection() {
  const form = usePolicyForm();

  return (
    <section className="space-y-4">
      <div className="p-4">
        <h3 className="font-medium">Policy</h3>

        <p className="text-sm text-muted-foreground">
          Basic information and where this policy applies.
        </p>
      </div>

      <Separator />
      <div className="p-4">
        <Card>
          <CardContent className="space-y-6 p-6">
            <form.Field
              name="name"
              children={(field) => (
                <div className="space-y-2">
                  <Label>Name</Label>

                  <Input
                    value={field.state.value}
                    placeholder="Production Organization"
                    onChange={(e) => field.handleChange(e.target.value)}
                  />
                </div>
              )}
            />

            <div className="grid grid-cols-2 gap-4">
              <form.Field
                name="scope"
                children={(field) => (
                  <div className="space-y-2">
                    <Label>Scope</Label>

                    <Select
                      value={field.state.value}
                      onValueChange={field.handleChange}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>

                      <SelectContent>
                        <SelectItem value="organization">
                          Organization
                        </SelectItem>

                        <SelectItem value="team">Team</SelectItem>

                        <SelectItem value="project">Project</SelectItem>

                        <SelectItem value="user">User</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                )}
              />

              <form.Field
                name="priority"
                children={(field) => (
                  <div className="space-y-2">
                    <Label>Priority</Label>

                    <Input
                      type="number"
                      value={field.state.value}
                      onChange={(e) =>
                        field.handleChange(Number(e.target.value))
                      }
                    />
                  </div>
                )}
              />
            </div>

            <form.Field
              name="targetId"
              children={(field) => (
                <div className="space-y-2">
                  <Label>Target</Label>

                  <Select
                    value={field.state.value}
                    onValueChange={field.handleChange}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select target" />
                    </SelectTrigger>

                    <SelectContent>
                      <SelectItem value="org">Default Organization</SelectItem>

                      <SelectItem value="backend">Backend Team</SelectItem>

                      <SelectItem value="marketing">Marketing Team</SelectItem>

                      <SelectItem value="project-alpha">
                        Project Alpha
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              )}
            />

            <form.Field
              name="enabled"
              children={(field) => (
                <div className="flex items-center justify-between rounded-lg border p-4">
                  <div>
                    <Label>Enabled</Label>

                    <p className="text-sm text-muted-foreground">
                      Enable this policy immediately.
                    </p>
                  </div>

                  <Switch
                    checked={field.state.value}
                    onCheckedChange={field.handleChange}
                  />
                </div>
              )}
            />
          </CardContent>
        </Card>
      </div>
    </section>
  );
}
