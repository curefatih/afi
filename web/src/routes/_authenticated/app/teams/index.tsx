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
    name: "Team Alpha",
    description: "This is Team Alpha, focused on project A.",
    previewMembers: [
      { name: "Alice", avatarUrl: "https://example.com/alice.jpg" },
      { name: "Bob", avatarUrl: "https://example.com/bob.jpg" },
      { name: "Charlie", avatarUrl: "https://example.com/charlie.jpg" },
    ],
    tags: ["Finance", "Engineering"],
    memberCount: 5,
  },
  {
    id: "2",
    name: "Team Beta",
    description: "This is Team Beta, focused on project B.",
    previewMembers: [
      { name: "David", avatarUrl: "https://example.com/david.jpg" },
      { name: "Eve", avatarUrl: "https://example.com/eve.jpg" },
      { name: "Frank", avatarUrl: "https://example.com/frank.jpg" },
    ],
    tags: ["Marketing", "Design"],
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
