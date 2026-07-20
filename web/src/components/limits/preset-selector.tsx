import { Card, CardContent } from "@/components/ui/card";
import { Separator } from "../ui/separator";

const presets = [
	{
		title: "Development",
		description: "Small limits for local testing.",
	},
	{
		title: "Production",
		description: "Recommended defaults.",
	},
	{
		title: "Enterprise",
		description: "High throughput.",
	},
];

export function PresetSelector() {
	return (
		<section className="space-y-4">
			<div className="p-4">
				<h3 className="font-medium">Presets</h3>
			</div>
			<Separator />

			<div className="grid gap-4 md:grid-cols-3 p-4">
				{presets.map((preset) => (
					<Card
						key={preset.title}
						className="cursor-pointer transition hover:border-primary"
					>
						<CardContent className="p-6">
							<h4 className="font-medium">{preset.title}</h4>

							<p className="mt-2 text-sm text-muted-foreground">
								{preset.description}
							</p>
						</CardContent>
					</Card>
				))}
			</div>
		</section>
	);
}
