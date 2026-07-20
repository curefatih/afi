import { useState } from "react";

import { PolicyCard } from "./policy-card";
import { PolicySheet } from "./policy-sheet";

export type Policy = {
	id: string;
	name: string;
	enabled: boolean;

	scope: "organization" | "team" | "project" | "user";

	priority: number;

	models: string[];

	requests: string;
	inputTokens: string;
	outputTokens: string;
};

const mockPolicies: Policy[] = [
	{
		id: "1",
		name: "Organization Default",
		enabled: true,
		scope: "organization",
		priority: 100,
		models: ["GPT-4o", "Claude Sonnet 4"],
		requests: "60 / minute",
		inputTokens: "2M / day",
		outputTokens: "1M / day",
	},
	{
		id: "2",
		name: "Backend Team",
		enabled: true,
		scope: "team",
		priority: 90,
		models: ["GPT-4.1", "GPT-4o Mini"],
		requests: "120 / minute",
		inputTokens: "10M / day",
		outputTokens: "5M / day",
	},
	{
		id: "3",
		name: "Marketing Project",
		enabled: true,
		scope: "project",
		priority: 75,
		models: ["Gemini 2.5 Pro"],
		requests: "500 / hour",
		inputTokens: "1M / day",
		outputTokens: "500K / day",
	},
	{
		id: "4",
		name: "John Doe",
		enabled: false,
		scope: "user",
		priority: 50,
		models: ["Claude Sonnet 4"],
		requests: "25 / minute",
		inputTokens: "100K / day",
		outputTokens: "50K / day",
	},
];

export function PolicyList() {
	const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);

	return (
		<>
			<div className="space-y-4">
				{mockPolicies.map((policy) => (
					<PolicyCard
						key={policy.id}
						policy={policy}
						onClick={() => setSelectedPolicy(policy)}
					/>
				))}
			</div>

			<PolicySheet
				policy={selectedPolicy}
				open={selectedPolicy !== null}
				onOpenChange={(open) => {
					if (!open) {
						setSelectedPolicy(null);
					}
				}}
			/>
		</>
	);
}
