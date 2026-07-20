import { Badge } from "../ui/badge";
import { Card, CardContent } from "../ui/card";
import { Checkbox } from "../ui/checkbox";
import { Label } from "../ui/label";
import { Separator } from "../ui/separator";
import { usePolicyForm } from "./policy-form-provider";

const providers = ["OpenAI", "Anthropic", "Google", "Mistral"];

const models = [
	"GPT-4o",
	"GPT-4.1",
	"Claude Sonnet 4",
	"Claude Opus 4",
	"Gemini 2.5 Pro",
	"DeepSeek R1",
];

const modalities = ["Chat", "Embedding", "Image", "Audio", "Video"];

export function MatchSection() {
	const form = usePolicyForm();

	return (
		<section className="space-y-6">
			<div className="p-4">
				<h3 className="font-medium">Matching</h3>

				<p className="text-sm text-muted-foreground">
					Choose which requests this policy applies to.
				</p>
			</div>

			<Separator />

			{/* Providers */}
			<div className="p-4">
				<Card>
					<CardContent className="space-y-4 p-6">
						<Label>Providers</Label>

						<div className="flex flex-wrap gap-2">
							{providers.map((provider) => (
								<Badge
									key={provider}
									variant="secondary"
									className="cursor-pointer px-3 py-2"
								>
									{provider}
								</Badge>
							))}
						</div>
					</CardContent>
				</Card>

				{/* Models */}

				<Card>
					<CardContent className="space-y-4 p-6">
						<Label>Models</Label>

						<div className="grid grid-cols-2 gap-3">
							{models.map((model) => (
								<form.Field key={model} name="models">
									{(field) => {
										const checked = field.state.value.includes(model);

										return (
											<Label className="flex items-center gap-3 rounded-md border p-3 font-normal">
												<Checkbox
													checked={checked}
													onCheckedChange={(value) => {
														if (value) {
															field.handleChange([...field.state.value, model]);
														} else {
															field.handleChange(
																field.state.value.filter((x) => x !== model),
															);
														}
													}}
												/>

												{model}
											</Label>
										);
									}}
								</form.Field>
							))}
						</div>
					</CardContent>
				</Card>

				{/* Modalities */}

				<Card>
					<CardContent className="space-y-4 p-6">
						<Label>Modalities</Label>

						<div className="grid grid-cols-3 gap-3">
							{modalities.map((modality) => (
								<form.Field key={modality} name="modalities">
									{(field) => {
										const checked = field.state.value.includes(modality);

										return (
											<Label className="flex items-center gap-3 rounded-md border p-3 font-normal">
												<Checkbox
													checked={checked}
													onCheckedChange={(value) => {
														if (value) {
															field.handleChange([
																...field.state.value,
																modality,
															]);
														} else {
															field.handleChange(
																field.state.value.filter((x) => x !== modality),
															);
														}
													}}
												/>

												{modality}
											</Label>
										);
									}}
								</form.Field>
							))}
						</div>
					</CardContent>
				</Card>
			</div>
		</section>
	);
}
