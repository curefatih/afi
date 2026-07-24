"use client";

import { useForm, useStore } from "@tanstack/react-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, CopyIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { z } from "zod";
import { environmentsQueryOptions } from "#/api/environment";
import {
	type ApiKey,
	createKeyMutationOptions,
	createOrgKeyMutationOptions,
	type KeyKind,
} from "#/api/keys";
import { InfoAlert } from "#/components/info-alert";
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
			environmentId: z.string(),
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
			saScope: (projectOnly ? "project" : "organization") as
				| "organization"
				| "project",
			projectId: defaultProjectId ?? activeOrg?.projects[0]?.id ?? "",
			environmentId: "",
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

			const environment_id = value.environmentId || undefined;

			if (projectOnly && defaultProjectId) {
				createProject.mutate(
					{
						projectId: defaultProjectId,
						name: value.name,
						environment_id,
					},
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
					environment_id:
						value.kind === "service_account" && value.saScope === "project"
							? environment_id
							: undefined,
				},
				{ onSuccess, onError },
			);
		},
	});

	// Subscribe so the environments query re-runs when kind/scope/project change.
	const kind = useStore(form.store, (s) => s.values.kind);
	const saScope = useStore(form.store, (s) => s.values.saScope);
	const projectId = useStore(form.store, (s) => s.values.projectId);

	const envProjectId = projectOnly
		? (defaultProjectId ?? "")
		: kind === "service_account" && saScope === "project"
			? projectId
			: "";
	const envs = useQuery(environmentsQueryOptions(orgId, envProjectId));
	const showEnv = !!envProjectId;

	useEffect(() => {
		if (!open) return;
		form.setFieldValue("kind", projectOnly ? "service_account" : defaultKind);
		form.setFieldValue("saScope", projectOnly ? "project" : "organization");
		form.setFieldValue(
			"projectId",
			defaultProjectId ?? activeOrg?.projects[0]?.id ?? "",
		);
		form.setFieldValue("environmentId", "");
		// Reset only when the sheet opens (or project binding changes), not on
		// every org.projects referential update — that cleared the env selection.
		// eslint-disable-next-line react-hooks/exhaustive-deps -- form API is stable
	}, [open, defaultKind, defaultProjectId, projectOnly]);

	const prevEnvProjectId = useRef(envProjectId);
	useEffect(() => {
		if (prevEnvProjectId.current === envProjectId) return;
		prevEnvProjectId.current = envProjectId;
		form.setFieldValue("environmentId", "");
		// eslint-disable-next-line react-hooks/exhaustive-deps -- only clear when project changes
	}, [envProjectId]);

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
							? "Copy the key before closing this sheet."
							: projectOnly
								? "Creates a project service-account key for automation."
								: "Personal keys belong to you. Service accounts are for automation (admin only)."}
					</SheetDescription>
					{createdKey ? (
						<InfoAlert>
							This secret is shown once. Store it securely before closing.
						</InfoAlert>
					) : null}
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

							{!projectOnly && kind === "service_account" ? (
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
									{saScope === "project" ? (
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
									) : null}
								</>
							) : null}

							{showEnv ? (
								<form.Field name="environmentId">
									{(field) => (
										<Field>
											<FieldLabel>Environment (optional)</FieldLabel>
											<Select
												value={field.state.value || "__none__"}
												onValueChange={(value) =>
													field.handleChange(
														!value || value === "__none__" ? "" : value,
													)
												}
											>
												<SelectTrigger className="w-full">
													<SelectValue placeholder="None" />
												</SelectTrigger>
												<SelectContent>
													<SelectItem value="__none__">None</SelectItem>
													{(envs.data ?? []).map((env) => (
														<SelectItem key={env.id} value={env.id}>
															{env.name} ({env.slug})
														</SelectItem>
													))}
												</SelectContent>
											</Select>
											{envs.isError ? (
												<p className="text-destructive text-xs">
													Could not load environments.
												</p>
											) : null}
											{!envs.isPending &&
											!envs.isError &&
											(envs.data?.length ?? 0) === 0 ? (
												<p className="text-muted-foreground text-xs">
													No environments on this project yet. Create one on the
													project page.
												</p>
											) : null}
										</Field>
									)}
								</form.Field>
							) : null}
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
