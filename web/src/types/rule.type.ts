export type LimitRule = {
	id: string;

	type:
		| "requests"
		| "input_tokens"
		| "output_tokens"
		| "total_tokens"
		| "spend"
		| "concurrency";

	operator: ">" | ">=" | "=";

	value: number;

	interval: "second" | "minute" | "hour" | "day" | "month";
};

export type Condition = {
	id: string;

	field:
		| "provider"
		| "model"
		| "api_key"
		| "region"
		| "environment"
		| "tag"
		| "metadata";

	operator: "equals" | "not_equals" | "contains" | "starts_with" | "in";

	value: string;
};
