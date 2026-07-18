import { teamsQueryOptions } from "#/api/team";
import TeamCard from "#/components/team-card";
import { useMutation } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";

export const Route = createFileRoute("/_authenticated/app/teams/")({
  staticData: {
    getTitle: () => "Team",
  },
  component: RouteComponent,
});

type Team = {
  id: string;
  name: string;
  description: string;
  previewMembers: {
    name: string;
    avatarUrl: string;
  }[];
  tags: string[];
  memberCount: number;
};

function RouteComponent() {
  const [teams, setTeams] = useState<Team[]>([]);

  const teamsMutation = useMutation({
    ...teamsQueryOptions(),
  });

  useEffect(() => {
    teamsMutation.mutate(undefined, {
      onSuccess(data, variables, onMutateResult, context) {
        setTeams(data);
      },
    });
  }, []);

  return (
    <div>
      <h1 className="scroll-m-20 text-2xl font-extrabold tracking-tight text-balance">
        Teams Directory
      </h1>
      <span className="mb-2 text-sm font-normal text-muted-foreground">
        Monitor velocity, managed projects, and core contributors across the
        organization.
      </span>

      <div className="teams flex flex-wrap gap-4 mt-2">
        {teams.map((team) => (
          <TeamCard
            key={team.id}
            id={team.id}
            name={team.name}
            description={team.description}
            previewMembers={team.previewMembers}
            memberCount={team.memberCount}
            tags={team.tags}
          />
        ))}
      </div>
    </div>
  );
}
