import { Separator } from "@/components/ui/separator";

import { AddConditionMenu } from "./add-condition-menu";
import { ConditionCard } from "./condition-card";

export function ConditionsSection() {
	return (
		<section className="space-y-6">
			<div className="flex items-center justify-between p-4">
				<div>
					<h3 className="font-medium">Conditions</h3>

					<p className="text-sm text-muted-foreground">
						Only apply this policy when all conditions match.
					</p>
				</div>

				<AddConditionMenu />
			</div>

			<Separator />

			<div className="p-4 flex flex-col gap-2">
				<ConditionCard onDelete={() => {}} />

				<ConditionCard onDelete={() => {}} />
			</div>
		</section>
	);
}
