import TeamCard from "#/components/team-card";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/app/teams/")({
  staticData: {
    getTitle: () => "Team",
  },
  component: RouteComponent,
});

const mockTeams = [
  {
    id: "1",
    previewMembers: [
      { name: "Alice", avatarUrl: "https://example.com/alice.jpg" },
      { name: "Bob", avatarUrl: "https://example.com/bob.jpg" },
      { name: "Charlie", avatarUrl: "https://example.com/charlie.jpg" },
    ],
    memberCount: 5,
  },
  {
    id: "2",
    previewMembers: [
      { name: "David", avatarUrl: "https://example.com/david.jpg" },
      { name: "Eve", avatarUrl: "https://example.com/eve.jpg" },
      { name: "Frank", avatarUrl: "https://example.com/frank.jpg" },
    ],
    memberCount: 3,
  },
];
function RouteComponent() {
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
        {mockTeams.map((team) => (
          <TeamCard
            key={team.id}
            id={team.id}
            previewMembers={team.previewMembers}
            memberCount={team.memberCount}
          />
        ))}
      </div>
    </div>
  );
}
