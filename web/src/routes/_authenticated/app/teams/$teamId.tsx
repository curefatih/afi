import { teamMembersQueryOptions, teamQueryOptions } from "#/api/team";
import { Card, CardContent, CardHeader } from "#/components/ui/card";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow
} from "#/components/ui/table";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/app/teams/$teamId")({
  staticData: {
    getTitle: () => "Team",
  },
  component: RouteComponent,
});

type TeamMember = {
  id: string;
  name: string;
  email: string;
  provider: string;
  external_id: string;
  created_at: string;
};
function RouteComponent() {
  const { teamId } = Route.useParams();
  console.log("teamId", teamId);

  const teamQuery = useQuery({
    ...teamQueryOptions(teamId),
  });

  const teamMemberQuery = useQuery({
    ...teamMembersQueryOptions(teamId),
  });

  return (
    <div>
      {teamQuery.isPending ? (
        "Getting team..."
      ) : (
        <>
          {teamQuery.isError ? (
            <div>An error occurred: {teamQuery.error.message}</div>
          ) : null}

          {teamQuery.isSuccess ? (
            <div className="">
              <h1 className="scroll-m-20 text-2xl font-extrabold tracking-tight text-balance">
                Team: {teamQuery.data.name}
              </h1>
              <span className="mb-2 text-sm font-normal text-muted-foreground">
                Overview and settings
              </span>

              <Card>
                <CardHeader>Members</CardHeader>
                <CardContent>
                  {teamMemberQuery.isPending ? (
                    "Getting members..."
                  ) : (
                    <>
                      {teamMemberQuery.isError ? (
                        <div>
                          An error occurred: {teamMemberQuery.error.message}
                        </div>
                      ) : null}

                      {teamMemberQuery.isSuccess ? (
                        <Table>
                          <TableCaption>
                            A list of team members
                          </TableCaption>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="w-[100px]">
                                Name
                              </TableHead>
                              <TableHead>Email</TableHead>
                              <TableHead>Provider</TableHead>
                              <TableHead className="text-right">
                                Created at
                              </TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {teamMemberQuery.data.map((member: TeamMember) => (
                              <TableRow key={member.id}>
                                <TableCell className="font-medium">
                                  {member.name}
                                </TableCell>
                                <TableCell>{member.email}</TableCell>
                                <TableCell>{member.provider}</TableCell>
                                <TableCell className="text-right text-muted-foreground">
                                  {member.created_at}
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                          <TableFooter>
                            <TableRow>
                              <TableCell colSpan={3}>Total</TableCell>
                              <TableCell className="text-right">
                                {teamMemberQuery.data.length}
                              </TableCell>
                            </TableRow>
                          </TableFooter>
                        </Table>
                      ) : null}
                    </>
                  )}
                </CardContent>
              </Card>
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}
