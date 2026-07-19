"use client";

import { useForm } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, CopyIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { z } from "zod";
import { type ApiKey, createKeyMutationOptions } from "#/api/keys";
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
import { useActiveOrg } from "#/state/organization-state";

const schema = z.object({
	name: z.string().min(1, "Name is required"),
	projectId: z.string().min(1, "Project is required"),
});

type CreateKeySheetProps = {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	defaultProjectId?: string;
	onCreated?: (key: ApiKey) => void;
};

export function CreateKeySheet({
	open,
	onOpenChange,
	defaultProjectId,
	onCreated,
}: CreateKeySheetProps) {
	const activeOrg = useActiveOrg();
	const queryClient = useQueryClient();
	const createMutation = useMutation(createKeyMutationOptions());
	const [createdKey, setCreatedKey] = useState<ApiKey | null>(null);
	const [copied, setCopied] = useState(false);

	const form = useForm({
		defaultValues: {
			name: "",
			projectId: defaultProjectId ?? activeOrg?.projects[0]?.id ?? "",
		},
		validators: {
			onChange: schema,
		},
		onSubmit: async ({ value }) => {
			createMutation.mutate(
				{
					projectId: value.projectId,
					name: value.name,
				},
				{
					onSuccess: (key) => {
						void queryClient.invalidateQueries({
							queryKey: ["projects", value.projectId, "keys"],
						});
						setCreatedKey(key);
						onCreated?.(key);
						toast.success("API key created");
					},
					onError: (error) => {
						toast.error(error.message || "Failed to create key");
					},
				},
			);
		},
	});

	const handleClose = (next: boolean) => {
		if (!next) {
			setCreatedKey(null);
			setCopied(false);
			form.reset();
		}
		onOpenChange(next);
	};

	return (
		<Sheet open={open} onOpenChange={handleClose}>
			<SheetContent>
				<SheetHeader>
					<SheetTitle>
						{createdKey ? "Copy your API key" : "Create API key"}
					</SheetTitle>
					<SheetDescription>
						{createdKey
							? "This secret is shown once. Store it securely before closing."
							: "Virtual API keys authenticate gateway requests for a project."}
					</SheetDescription>
				</SheetHeader>

				{createdKey ? (
					<div className="flex flex-1 flex-col gap-4 px-4">
						<Field>
							<FieldLabel>Key name</FieldLabel>
							<Input readOnly value={createdKey.name} />
						</Field>
						<Field>
							<FieldLabel>Secret</FieldLabel>
							<div className="flex gap-2">
								<Input
									readOnly
									value={createdKey.key}
									className="font-mono text-xs"
								/>
								<Button
									type="button"
									variant="outline"
									size="icon"
									onClick={async () => {
										await navigator.clipboard.writeText(createdKey.key);
										setCopied(true);
										toast.success("Copied to clipboard");
										setTimeout(() => setCopied(false), 1500);
									}}
								>
									{copied ? <CheckIcon /> : <CopyIcon />}
								</Button>
							</div>
						</Field>
						<SheetFooter>
							<Button type="button" onClick={() => handleClose(false)}>
								Done
							</Button>
						</SheetFooter>
					</div>
				) : (
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
										<FieldLabel htmlFor="key-name">Name</FieldLabel>
										<Input
											id="key-name"
											placeholder="Playground"
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
							<form.Field name="projectId">
								{(field) => (
									<Field>
										<FieldLabel>Project</FieldLabel>
										<Select
											value={field.state.value}
											onValueChange={(value) => field.handleChange(value ?? "")}
											disabled={!!defaultProjectId}
										>
											<SelectTrigger className="w-full">
												<SelectValue placeholder="Select a project" />
											</SelectTrigger>
											<SelectContent>
												{(activeOrg?.projects ?? []).map((project) => (
													<SelectItem key={project.id} value={project.id}>
														{project.name}
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
								onClick={() => handleClose(false)}
							>
								Cancel
							</Button>
							<Button type="submit" disabled={createMutation.isPending}>
								{createMutation.isPending ? "Creating…" : "Create key"}
							</Button>
						</SheetFooter>
					</form>
				)}
			</SheetContent>
		</Sheet>
	);
}
