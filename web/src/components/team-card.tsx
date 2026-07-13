import { Card, CardContent, CardFooter } from "./ui/card";
import {
  Avatar,
  AvatarFallback,
  AvatarGroup,
  AvatarGroupCount,
  AvatarImage,
} from "./ui/avatar";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { Link } from "@tanstack/react-router";
import { Separator } from "./ui/separator";

type PreviewMember = {
  name: string;
  avatarUrl: string;
};

type TeamCardProps = {
  id: string;
  name: string;
  description: string;
  previewMembers: PreviewMember[];
  memberCount: number;
  tags?: string[];
};

export default function TeamCard({
  id,
  name,
  description,
  previewMembers,
  memberCount,
  tags,
}: TeamCardProps) {
  return (
    <Card className="team-card w-72 border">
      <CardContent className="p-4">
        <h2 className="text-lg font-semibold">{name}</h2>
        <p className="text-sm text-muted-foreground">{description}</p>
        <div className="mt-2">
          {tags?.map((tag, index) => (
            <Badge key={index} className="mr-1">
              {tag}
            </Badge>
          ))}
        </div>
      </CardContent>
      <CardFooter className="flex items-center justify-between p-4">
        <AvatarGroup className="grayscale">
          {previewMembers.map((member, index) => (
            <Avatar key={index}>
              <AvatarImage src={member.avatarUrl} alt={member.name} />
              <AvatarFallback>{member.name.charAt(0)}</AvatarFallback>
            </Avatar>
          ))}
          <AvatarGroupCount>{memberCount}</AvatarGroupCount>
        </AvatarGroup>
        <Button variant="outline" size="sm">
          <Link size="sm" to={`/app/teams/${id}`} className="mr-2">
            View Team
          </Link>
        </Button>
      </CardFooter>
    </Card>
  );
}
