"use client";

import { useForm } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { z } from "zod";
import { createProjectMutationOptions } from "#/api/project";
import { Button } from "#/components/ui/button";
import {
	Field,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
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
	team_id: z.string().min(1, "Team is required"),
});

type CreateProjectSheetProps = {
	open: boolean;
	onOpenChange: (open: boolean) => void;
};

export function CreateProjectSheet({
	open,
	onOpenChange,
}: CreateProjectSheetProps) {
	const activeOrg = useActiveOrg();
	const { upsertProject } = useOrgActions();
	const queryClient = useQueryClient();
	const createMutation = useMutation(createProjectMutationOptions());

	const form = useForm({
		defaultValues: {
			name: "",
			team_id: activeOrg?.teams[0]?.id ?? "",
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
					team_id: value.team_id,
				},
				{
					onSuccess: (project) => {
						upsertProject(activeOrg.id, project);
						void queryClient.invalidateQueries({
							queryKey: ["organizations", activeOrg.id, "projects"],
						});
						toast.success("Project created");
						form.reset();
						onOpenChange(false);
					},
					onError: (error) => {
						toast.error(error.message || "Failed to create project");
					},
				},
			);
		},
	});

	return (
		<Sheet open={open} onOpenChange={onOpenChange}>
			<SheetContent>
				<SheetHeader>
					<SheetTitle>Create project</SheetTitle>
					<SheetDescription>
						Projects group virtual API keys and provider assignments for a team.
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
									<FieldLabel htmlFor="project-name">Name</FieldLabel>
									<Input
										id="project-name"
										placeholder="Production"
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
						<form.Field name="team_id">
							{(field) => (
								<Field>
									<FieldLabel>Team</FieldLabel>
									<Select
										value={field.state.value}
										onValueChange={(value) => field.handleChange(value ?? "")}
									>
										<SelectTrigger className="w-full">
											<SelectValue placeholder="Select a team" />
										</SelectTrigger>
										<SelectContent>
											{(activeOrg?.teams ?? []).map((team) => (
												<SelectItem key={team.id} value={team.id}>
													{team.name}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
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
							{createMutation.isPending ? "Creating…" : "Create project"}
						</Button>
					</SheetFooter>
				</form>
			</SheetContent>
		</Sheet>
	);
}
