"use client";

import { useForm, useStore } from "@tanstack/react-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { toast } from "sonner";
import { z } from "zod";
import { environmentsQueryOptions } from "#/api/environment";
import { createSigningKeyMutationOptions } from "#/api/signing-keys";
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
import { Textarea } from "#/components/ui/textarea";
import { useActiveOrg } from "#/state/organization-state";

type CreateSigningKeySheetProps = {
	open: boolean;
	onOpenChange: (open: boolean) => void;
};

function suggestKeyId() {
	const bytes = new Uint8Array(4);
	crypto.getRandomValues(bytes);
	const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join(
		"",
	);
	return `svc-${hex}`;
}

export function CreateSigningKeySheet({
	open,
	onOpenChange,
}: CreateSigningKeySheetProps) {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const queryClient = useQueryClient();
	const create = useMutation(createSigningKeyMutationOptions());

	const schema = z
		.object({
			name: z.string().min(1, "Name is required"),
			keyId: z
				.string()
				.min(1, "Key ID is required")
				.regex(
					/^[a-zA-Z0-9][a-zA-Z0-9._:-]*$/,
					"Use letters, numbers, and . _ : -",
				),
			scope: z.enum(["organization", "project"]),
			projectId: z.string(),
			environmentId: z.string(),
			publicKeyPem: z
				.string()
				.min(1, "Public key PEM is required")
				.refine(
					(v) => v.includes("BEGIN PUBLIC KEY") || v.includes("BEGIN "),
					"Paste a PEM-encoded public key",
				),
		})
		.superRefine((val, ctx) => {
			if (val.scope === "project" && !val.projectId) {
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
			keyId: suggestKeyId(),
			scope: "organization" as "organization" | "project",
			projectId: activeOrg?.projects[0]?.id ?? "",
			environmentId: "",
			publicKeyPem: "",
		},
		validators: {
			onChange: schema,
		},
		onSubmit: async ({ value }) => {
			create.mutate(
				{
					orgId,
					key_id: value.keyId.trim(),
					name: value.name.trim(),
					algorithm: "ed25519",
					public_key_pem: value.publicKeyPem.trim(),
					project_id:
						value.scope === "project"
							? value.projectId || undefined
							: undefined,
					environment_id:
						value.scope === "project"
							? value.environmentId || undefined
							: undefined,
				},
				{
					onSuccess: () => {
						void queryClient.invalidateQueries({
							queryKey: ["organizations", orgId, "signing-keys"],
						});
						toast.success("Signing key created");
						form.reset();
						onOpenChange(false);
					},
					onError: (error: Error) => {
						toast.error(error.message || "Failed to create signing key");
					},
				},
			);
		},
	});

	const scope = useStore(form.store, (s) => s.values.scope);
	const projectId = useStore(form.store, (s) => s.values.projectId);
	const envProjectId = scope === "project" ? projectId : "";
	const envs = useQuery(environmentsQueryOptions(orgId, envProjectId));

	const firstProjectId = activeOrg?.projects[0]?.id;
	useEffect(() => {
		if (!open) return;
		form.setFieldValue("name", "");
		form.setFieldValue("keyId", suggestKeyId());
		form.setFieldValue("scope", "organization");
		form.setFieldValue("projectId", firstProjectId ?? "");
		form.setFieldValue("environmentId", "");
		form.setFieldValue("publicKeyPem", "");
	}, [open, firstProjectId, form.setFieldValue]);

	const prevEnvProjectId = useRef(envProjectId);
	useEffect(() => {
		if (prevEnvProjectId.current === envProjectId) return;
		prevEnvProjectId.current = envProjectId;
		form.setFieldValue("environmentId", "");
	}, [envProjectId, form.setFieldValue]);

	const handleClose = (next: boolean) => {
		if (!next) form.reset();
		onOpenChange(next);
	};

	return (
		<Sheet open={open} onOpenChange={handleClose}>
			<SheetContent>
				<SheetHeader>
					<SheetTitle>Register signing key</SheetTitle>
					<SheetDescription>
						Upload the Ed25519 public key your service will use to sign gateway
						requests. Keep the private key on your side.
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
									<FieldLabel htmlFor="signing-key-name">Name</FieldLabel>
									<Input
										id="signing-key-name"
										placeholder="Payments worker"
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

						<form.Field name="keyId">
							{(field) => (
								<Field>
									<FieldLabel htmlFor="signing-key-id">Key ID</FieldLabel>
									<Input
										id="signing-key-id"
										placeholder="svc-payments"
										className="font-mono text-sm"
										value={field.state.value}
										onChange={(e) => field.handleChange(e.target.value)}
										onBlur={field.handleBlur}
									/>
									<p className="text-muted-foreground text-xs">
										Used as the RFC 9421 signature keyid parameter.
									</p>
									{!field.state.meta.isValid ? (
										<FieldError errors={field.state.meta.errors} />
									) : null}
								</Field>
							)}
						</form.Field>

						<form.Field name="scope">
							{(field) => (
								<Field>
									<FieldLabel>Scope</FieldLabel>
									<Select
										value={field.state.value}
										onValueChange={(value) =>
											field.handleChange(
												(value as "organization" | "project") ?? "organization",
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

						{scope === "project" ? (
							<>
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
										</Field>
									)}
								</form.Field>
							</>
						) : null}

						<form.Field name="publicKeyPem">
							{(field) => (
								<Field>
									<FieldLabel htmlFor="signing-key-pem">
										Public key (PEM)
									</FieldLabel>
									<Textarea
										id="signing-key-pem"
										placeholder={
											"-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
										}
										className="min-h-36 font-mono text-xs"
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
							onClick={() => handleClose(false)}
						>
							Cancel
						</Button>
						<Button type="submit" disabled={create.isPending || !orgId}>
							{create.isPending ? "Creating…" : "Register key"}
						</Button>
					</SheetFooter>
				</form>
			</SheetContent>
		</Sheet>
	);
}
