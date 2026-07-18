import { Sheet, SheetContent } from "@/components/ui/sheet";
import { BehaviorSection } from "./behaviour-section";
import { LimitsSection } from "./limits-section";
import { MatchSection } from "./match-section";
import type { Policy } from "./policy-list";
import { PolicySection } from "./policy-section";
import { PresetSelector } from "./preset-selector";
import { SummaryPanel } from "./summary-panel";
import { useForm } from "@tanstack/react-form";
import { PolicyFormProvider } from "./policy-form-provider";
import { ConditionsSection } from "./conditions-section";
import { Separator } from "@base-ui/react";

type PolicySheetProps = {
  open: boolean;
  policy: Policy | null;
  onOpenChange(open: boolean): void;
};

export function PolicySheet({ open, policy, onOpenChange }: PolicySheetProps) {
  const isEditing = policy !== null;

  // const form = useForm({
  //   defaultValues: {
  //     name: "",
  //     enabled: true,

  //     scope: "organization",

  //     targetId: "",

  //     priority: 100,

  //     models: [] as string[],

  //     modalities: ["text"],

  //     requestLimit: "",

  //     requestInterval: "minute",

  //     inputTokens: "",

  //     outputTokens: "",

  //     tokenInterval: "day",

  //     action: "reject",

  //     fallbackModel: "",
  //   },

  //   onSubmit: async ({ value }) => {
  //     console.log(value);
  //   },
  // });

  return (
    <PolicyFormProvider>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent className="!min-w-[50vw] max-w-[1600px] overflow-y-auto">
          <div className="flex flex-col">
            <div className="space-y-10">
              <PolicySection />

              <MatchSection />

              <ConditionsSection />

              <PresetSelector />

              <LimitsSection />

              <BehaviorSection />
              
            </div>

            <SummaryPanel />
          </div>
        </SheetContent>
      </Sheet>
    </PolicyFormProvider>
  );
}
