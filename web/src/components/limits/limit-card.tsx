import { useState } from "react";
import { Card, CardContent } from "../ui/card";
import { Switch } from "../ui/switch";

type LimitCardProps = {
	title: string;
	description: string;
	children: React.ReactNode;
};

export function LimitCard({ title, description, children }: LimitCardProps) {
	const [enabled, setEnabled] = useState(false);

	return (
		<Card>
			<CardContent className="space-y-6 p-6">
				<div className="flex items-start justify-between">
					<div>
						<h4 className="font-medium">{title}</h4>

						<p className="text-sm text-muted-foreground">{description}</p>
					</div>

					<Switch checked={enabled} onCheckedChange={setEnabled} />
				</div>

				{enabled && children}
			</CardContent>
		</Card>
	);
}
