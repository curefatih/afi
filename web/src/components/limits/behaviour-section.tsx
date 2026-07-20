import { Card, CardContent } from "../ui/card";
import { Label } from "../ui/label";
import { RadioGroup, RadioGroupItem } from "../ui/radio-group";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "../ui/select";
import { Separator } from "../ui/separator";
import { usePolicyForm } from "./policy-form-provider";

export function BehaviorSection() {
	const _form = usePolicyForm();

	return (
		<section className="space-y-4">
			<div className="p-4">
				<h3 className="font-medium">Behavior</h3>

				<p className="text-sm text-muted-foreground">
					Configure what happens when a policy is exceeded.
				</p>
			</div>

			<Separator />

			<div className="p-4">
				<Card>
					<CardContent className="space-y-6 p-6">
						<RadioGroup defaultValue="reject">
							<div className="flex items-center space-x-3">
								<RadioGroupItem value="reject" id="reject" />
								<Label htmlFor="reject">Reject request (429)</Label>
							</div>

							<div className="flex items-center space-x-3">
								<RadioGroupItem value="queue" id="queue" />
								<Label htmlFor="queue">Queue request</Label>
							</div>

							<div className="flex items-center space-x-3">
								<RadioGroupItem value="fallback" id="fallback" />
								<Label htmlFor="fallback">Fallback model</Label>
							</div>
						</RadioGroup>

						<div className="space-y-2">
							<Label>Fallback Model</Label>

							<Select>
								<SelectTrigger>
									<SelectValue placeholder="Select model" />
								</SelectTrigger>

								<SelectContent>
									<SelectItem value="gpt-4o-mini">GPT-4o Mini</SelectItem>

									<SelectItem value="claude-sonnet">Claude Sonnet</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</CardContent>
				</Card>
			</div>
		</section>
	);
}
