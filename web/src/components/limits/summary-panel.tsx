import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { usePolicyForm } from "./policy-form-provider";

export function SummaryPanel() {
  const form = usePolicyForm();

  return (
    <div className="p-4">
  
      <Card className="sticky top-6">
        <CardHeader>
          <CardTitle>Summary</CardTitle>
        </CardHeader>

        <CardContent className="space-y-4 text-sm">
          <SummaryItem label="Scope" value="Team" />

          <SummaryItem label="Models" value="3 selected" />

          <SummaryItem label="Modalities" value="Chat, Embedding" />

          <SummaryItem label="Requests" value="100 / minute" />

          <SummaryItem label="Input" value="2M / day" />

          <SummaryItem label="Output" value="1M / day" />
        </CardContent>
      </Card>
    </div>
  );
}

function SummaryItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-muted-foreground">{label}</span>

      <span>{value}</span>
    </div>
  );
}
