"use client";

import { useForm } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, CopyIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { z } from "zod";
import {
	type ApiKey,
	createKeyMutationOptions,
	createOrgKeyMutationOptions,
	type KeyKind,
} from "#/api/keys";
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

type CreateKeySheetProps = {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	/** When set, forces project SA create via project endpoint. */
	defaultProjectId?: string;
	/** Default tab kind when using org endpoint. */
	defaultKind?: KeyKind;
	isOrgAdmin?: boolean;
	onCreated?: (key: ApiKey) => void;
};

export function CreateKeySheet({
	open,
	onOpenChange,
	defaultProjectId,
	defaultKind = "personal",
	isOrgAdmin = false,
	onCreated,
}: CreateKeySheetProps) {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const queryClient = useQueryClient();
	const createOrg = useMutation(createOrgKeyMutationOptions());
	const createProject = useMutation(createKeyMutationOptions());
	const [createdKey, setCreatedKey] = useState<ApiKey | null>(null);
	const [copied, setCopied] = useState(false);

	const projectOnly = !!defaultProjectId;

	const schema = z
		.object({
			name: z.string().min(1, "Name is required"),
			kind: z.enum(["personal", "service_account"]),
			saScope: z.enum(["organization", "project"]),
			projectId: z.string(),
		})
		.superRefine((val, ctx) => {
			if (projectOnly) return;
			if (
				val.kind === "service_account" &&
				val.saScope === "project" &&
				!val.projectId
			) {
				ctx.addIssue({
					code: "custom",
					message: "Project is required",
					path: ["projectId"],
				});
			}
		});

	const form = useForm({
		defaultValues: {
			name: "",
			kind: projectOnly ? ("service_account" as KeyKind) : defaultKind,
			saScope: "organization" as "organization" | "project",
			projectId: defaultProjectId ?? activeOrg?.projects[0]?.id ?? "",
		},
		validators: {
			onChange: schema,
		},
		onSubmit: async ({ value }) => {
			const onSuccess = (key: ApiKey) => {
				void queryClient.invalidateQueries({
					queryKey: ["organizations", orgId, "keys"],
				});
				if (key.project_id) {
					void queryClient.invalidateQueries({
						queryKey: ["projects", key.project_id, "keys"],
					});
				}
				setCreatedKey(key);
				onCreated?.(key);
				toast.success("API key created");
			};
			const onError = (error: Error) => {
				toast.error(error.message || "Failed to create key");
			};

			if (projectOnly && defaultProjectId) {
				createProject.mutate(
					{ projectId: defaultProjectId, name: value.name },
					{ onSuccess, onError },
				);
				return;
			}

			createOrg.mutate(
				{
					orgId,
					name: value.name,
					kind: value.kind,
					project_id:
						value.kind === "service_account" && value.saScope === "project"
							? value.projectId
							: undefined,
				},
				{ onSuccess, onError },
			);
		},
	});

	useEffect(() => {
		if (!open) return;
		form.setFieldValue("kind", projectOnly ? "service_account" : defaultKind);
		form.setFieldValue(
			"projectId",
			defaultProjectId ?? activeOrg?.projects[0]?.id ?? "",
		);
	}, [
		open,
		defaultKind,
		defaultProjectId,
		projectOnly,
		activeOrg?.projects,
		form,
	]);

	const handleClose = (next: boolean) => {
		if (!next) {
			setCreatedKey(null);
			setCopied(false);
			form.reset();
		}
		onOpenChange(next);
	};

	const pending = createOrg.isPending || createProject.isPending;

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
							: projectOnly
								? "Creates a project service-account key for automation."
								: "Personal keys belong to you. Service accounts are for automation (admin only)."}
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
										if (!createdKey.key) return;
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
											placeholder="My laptop"
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

							{!projectOnly ? (
								<form.Field name="kind">
									{(field) => (
										<Field>
											<FieldLabel>Kind</FieldLabel>
											<Select
												value={field.state.value}
												onValueChange={(value) =>
													field.handleChange((value as KeyKind) ?? "personal")
												}
											>
												<SelectTrigger className="w-full">
													<SelectValue />
												</SelectTrigger>
												<SelectContent>
													<SelectItem value="personal">Personal</SelectItem>
													{isOrgAdmin ? (
														<SelectItem value="service_account">
															Service account
														</SelectItem>
													) : null}
												</SelectContent>
											</Select>
										</Field>
									)}
								</form.Field>
							) : null}

							<form.Subscribe selector={(s) => s.values.kind}>
								{(kind) =>
									!projectOnly && kind === "service_account" ? (
										<>
											<form.Field name="saScope">
												{(field) => (
													<Field>
														<FieldLabel>Scope</FieldLabel>
														<Select
															value={field.state.value}
															onValueChange={(value) =>
																field.handleChange(
																	(value as "organization" | "project") ??
																		"organization",
																)
															}
														>
															<SelectTrigger className="w-full">
																<SelectValue />
															</SelectTrigger>
															<SelectContent>
																<SelectItem value="organization">
																	Organization-wide
																</SelectItem>
																<SelectItem value="project">Project</SelectItem>
															</SelectContent>
														</Select>
													</Field>
												)}
											</form.Field>
											<form.Subscribe selector={(s) => s.values.saScope}>
												{(saScope) =>
													saScope === "project" ? (
														<form.Field name="projectId">
															{(field) => (
																<Field>
																	<FieldLabel>Project</FieldLabel>
																	<Select
																		value={field.state.value}
																		onValueChange={(value) =>
																			field.handleChange(value ?? "")
																		}
																	>
																		<SelectTrigger className="w-full">
																			<SelectValue placeholder="Select a project" />
																		</SelectTrigger>
																		<SelectContent>
																			{(activeOrg?.projects ?? []).map(
																				(project) => (
																					<SelectItem
																						key={project.id}
																						value={project.id}
																					>
																						{project.name}
																					</SelectItem>
																				),
																			)}
																		</SelectContent>
																	</Select>
																	{!field.state.meta.isValid ? (
																		<FieldError
																			errors={field.state.meta.errors}
																		/>
																	) : null}
																</Field>
															)}
														</form.Field>
													) : null
												}
											</form.Subscribe>
										</>
									) : null
								}
							</form.Subscribe>
						</FieldGroup>
						<SheetFooter>
							<Button
								type="button"
								variant="outline"
								onClick={() => handleClose(false)}
							>
								Cancel
							</Button>
							<Button type="submit" disabled={pending || !orgId}>
								{pending ? "Creating…" : "Create key"}
							</Button>
						</SheetFooter>
					</form>
				)}
			</SheetContent>
		</Sheet>
	);
}
