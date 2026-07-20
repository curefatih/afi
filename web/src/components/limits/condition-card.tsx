import { Trash2 } from "lucide-react";
import { Button } from "#/components/ui/button";
import { Card, CardContent } from "#/components/ui/card";
import { Input } from "#/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { Label } from "../ui/label";

type Props = {
	onDelete(): void;
};

export function ConditionCard({ onDelete }: Props) {
	return (
		<Card>
			<CardContent className="space-y-6 p-6">
				<div className="flex items-center justify-between">
					<h4 className="font-medium">Condition</h4>

					<Button size="icon" variant="ghost" onClick={onDelete}>
						<Trash2 className="h-4 w-4" />
					</Button>
				</div>

				<div className="grid grid-cols-3 gap-4">
					<div className="space-y-2">
						<Label>Field</Label>

						<Select>
							<SelectTrigger>
								<SelectValue />
							</SelectTrigger>

							<SelectContent>
								<SelectItem value="provider">Provider</SelectItem>

								<SelectItem value="model">Model</SelectItem>

								<SelectItem value="region">Region</SelectItem>

								<SelectItem value="environment">Environment</SelectItem>

								<SelectItem value="api_key">API Key</SelectItem>

								<SelectItem value="tag">Tag</SelectItem>

								<SelectItem value="metadata">Metadata</SelectItem>
							</SelectContent>
						</Select>
					</div>

					<div className="space-y-2">
						<Label>Operator</Label>

						<Select>
							<SelectTrigger>
								<SelectValue />
							</SelectTrigger>

							<SelectContent>
								<SelectItem value="equals">Equals</SelectItem>

								<SelectItem value="contains">Contains</SelectItem>

								<SelectItem value="in">In</SelectItem>

								<SelectItem value="starts_with">Starts With</SelectItem>
							</SelectContent>
						</Select>
					</div>

					<div className="space-y-2">
						<Label>Value</Label>

						<Input placeholder="production" />
					</div>
				</div>
			</CardContent>
		</Card>
	);
}
