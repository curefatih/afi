import { Link } from "@tanstack/react-router";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import {
	Card,
	CardContent,
	CardFooter,
	CardHeader,
	CardTitle,
} from "./ui/card";

type TeamCardProps = {
	id: string;
	name: string;
	description: string;
	previewMembers: { name: string; avatarUrl: string }[];
	memberCount: number;
	tags?: string[];
};

export default function TeamCard({
	id,
	name,
	description,
	tags,
}: TeamCardProps) {
	return (
		<Card className="w-72">
			<CardHeader>
				<CardTitle className="text-lg">{name}</CardTitle>
				<p className="font-mono text-xs text-muted-foreground">{description}</p>
				{tags && tags.length > 0 ? (
					<div className="mt-2 flex flex-wrap gap-1">
						{tags.map((tag) => (
							<Badge key={tag} variant="secondary">
								{tag}
							</Badge>
						))}
					</div>
				) : null}
			</CardHeader>
			<CardContent />
			<CardFooter>
				<Button
					variant="outline"
					size="sm"
					render={<Link to="/app/teams/$teamId" params={{ teamId: id }} />}
				>
					View team
				</Button>
			</CardFooter>
		</Card>
	);
}
