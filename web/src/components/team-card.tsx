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

type PreviewMember = {
  name: string;
  avatarUrl: string;
};

type TeamCardProps = {
  id: string;
  previewMembers: PreviewMember[];
  memberCount: number;
};

export default function TeamCard({
  id,
  previewMembers,
  memberCount,
}: TeamCardProps) {
  return (
    <Card className="team-card w-64">
      <CardContent className="p-4">
        <h2>Team 1</h2>
        <p>This is the first team.</p>
        <Badge variant="secondary" className="mt-2">
          5 Members
        </Badge>
      </CardContent>
      <CardFooter className="flex items-center justify-between p-4">
        <AvatarGroup className="grayscale">
          <Avatar>
            <AvatarImage src="https://github.com/shadcn.png" alt="@shadcn" />
            <AvatarFallback>CN</AvatarFallback>
          </Avatar>
          <Avatar>
            <AvatarImage
              src="https://github.com/maxleiter.png"
              alt="@maxleiter"
            />
            <AvatarFallback>LR</AvatarFallback>
          </Avatar>
          <Avatar>
            <AvatarImage
              src="https://github.com/evilrabbit.png"
              alt="@evilrabbit"
            />
            <AvatarFallback>ER</AvatarFallback>
          </Avatar>
          <AvatarGroupCount>+3</AvatarGroupCount>
        </AvatarGroup>
        <Button variant="outline" size="sm">
          <Link
            size="sm"
            to={`/app/teams/${id}`}
            className="mr-2"
          >
            View Team
          </Link>
        </Button>
      </CardFooter>
    </Card>
  );
}
