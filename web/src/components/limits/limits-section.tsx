import { Separator } from "@/components/ui/separator";

import { AddLimitMenu } from "./add-limit-menu";
import { LimitRuleCard } from "./limit-rule-card";
import { usePolicyForm } from "./policy-form-provider";

export function LimitsSection() {
  const form = usePolicyForm();

  return (
    <section className="space-y-6">
      <div className="flex items-center justify-between p-4">
        <div>
          <h3 className="font-medium">Limits</h3>

          <p className="text-sm text-muted-foreground">
            Configure one or more limits.
          </p>
        </div>

        <AddLimitMenu />
      </div>

      <Separator />

      <div className="p-4 flex flex-col gap-4">
        <LimitRuleCard title="Request Count" onDelete={() => {}} />

        <LimitRuleCard title="Input Tokens" onDelete={() => {}} />
      </div>
    </section>
  );
}
