"use client";

import { useForm } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { z } from "zod";
import { createTeamMutationOptions } from "#/api/team";
import { Button } from "#/components/ui/button";
import {
	Field,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import {
	Sheet,
	SheetContent,
	SheetDescription,
	SheetFooter,
	SheetHeader,
	SheetTitle,
} from "#/components/ui/sheet";
import { useActiveOrg, useOrgActions } from "#/state/organization-state";

const schema = z.object({
	name: z.string().min(1, "Name is required"),
});

type CreateTeamSheetProps = {
	open: boolean;
	onOpenChange: (open: boolean) => void;
};

export function CreateTeamSheet({ open, onOpenChange }: CreateTeamSheetProps) {
	const activeOrg = useActiveOrg();
	const { upsertTeam } = useOrgActions();
	const queryClient = useQueryClient();
	const createMutation = useMutation(createTeamMutationOptions());

	const form = useForm({
		defaultValues: {
			name: "",
		},
		validators: {
			onChange: schema,
		},
		onSubmit: async ({ value }) => {
			if (!activeOrg) return;
			createMutation.mutate(
				{
					orgId: activeOrg.id,
					name: value.name,
				},
				{
					onSuccess: (team) => {
						upsertTeam(activeOrg.id, team);
						void queryClient.invalidateQueries({
							queryKey: ["organizations", activeOrg.id, "teams"],
						});
						toast.success("Team created");
						form.reset();
						onOpenChange(false);
					},
					onError: (error) => {
						toast.error(error.message || "Failed to create team");
					},
				},
			);
		},
	});

	return (
		<Sheet open={open} onOpenChange={onOpenChange}>
			<SheetContent>
				<SheetHeader>
					<SheetTitle>Create team</SheetTitle>
					<SheetDescription>
						Teams group members and own projects within the organization.
					</SheetDescription>
				</SheetHeader>
				<form
					className="flex flex-1 flex-col gap-4 px-4"
					onSubmit={(e) => {
						e.preventDefault();
						void form.handleSubmit();
					}}
				>
					<FieldGroup>
						<form.Field name="name">
							{(field) => (
								<Field>
									<FieldLabel htmlFor="team-name">Name</FieldLabel>
									<Input
										id="team-name"
										placeholder="Platform"
										value={field.state.value}
										onChange={(e) => field.handleChange(e.target.value)}
										onBlur={field.handleBlur}
									/>
									{!field.state.meta.isValid ? (
										<FieldError errors={field.state.meta.errors} />
									) : null}
								</Field>
							)}
						</form.Field>
					</FieldGroup>
					<SheetFooter>
						<Button
							type="button"
							variant="outline"
							onClick={() => onOpenChange(false)}
						>
							Cancel
						</Button>
						<Button type="submit" disabled={createMutation.isPending}>
							{createMutation.isPending ? "Creating…" : "Create team"}
						</Button>
					</SheetFooter>
				</form>
			</SheetContent>
		</Sheet>
	);
}
