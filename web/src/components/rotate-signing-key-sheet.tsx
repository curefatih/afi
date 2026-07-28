"use client";

import { useForm } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { toast } from "sonner";
import { z } from "zod";
import {
	rotateSigningKeyMutationOptions,
	type SigningKey,
} from "#/api/signing-keys";
import { Button } from "#/components/ui/button";
import {
	Field,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "#/components/ui/field";
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

type RotateSigningKeySheetProps = {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	signingKey: SigningKey | null;
};

export function RotateSigningKeySheet({
	open,
	onOpenChange,
	signingKey,
}: RotateSigningKeySheetProps) {
	const activeOrg = useActiveOrg();
	const orgId = activeOrg?.id ?? "";
	const queryClient = useQueryClient();
	const rotate = useMutation(rotateSigningKeyMutationOptions());

	const schema = z.object({
		publicKeyPem: z
			.string()
			.min(1, "Public key PEM is required")
			.refine(
				(v) => v.includes("BEGIN PUBLIC KEY") || v.includes("BEGIN "),
				"Paste a PEM-encoded public key",
			),
	});

	const form = useForm({
		defaultValues: {
			publicKeyPem: "",
		},
		validators: {
			onChange: schema,
		},
		onSubmit: async ({ value }) => {
			if (!signingKey) return;
			rotate.mutate(
				{
					signingKeyId: signingKey.id,
					public_key_pem: value.publicKeyPem.trim(),
				},
				{
					onSuccess: () => {
						void queryClient.invalidateQueries({
							queryKey: ["organizations", orgId, "signing-keys"],
						});
						toast.success("Signing key rotated");
						form.reset();
						onOpenChange(false);
					},
					onError: (error: Error) => {
						toast.error(error.message || "Failed to rotate signing key");
					},
				},
			);
		},
	});

	useEffect(() => {
		if (!open) return;
		form.setFieldValue("publicKeyPem", "");
	}, [open, form.setFieldValue]);

	const handleClose = (next: boolean) => {
		if (!next) form.reset();
		onOpenChange(next);
	};

	return (
		<Sheet open={open} onOpenChange={handleClose}>
			<SheetContent>
				<SheetHeader>
					<SheetTitle>Rotate public key</SheetTitle>
					<SheetDescription>
						Replace the public key for{" "}
						<span className="font-medium text-foreground">
							{signingKey?.name ?? "this key"}
						</span>
						. The key ID stays the same; update your service&apos;s private key
						to match.
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
						<form.Field name="publicKeyPem">
							{(field) => (
								<Field>
									<FieldLabel htmlFor="rotate-signing-key-pem">
										New public key (PEM)
									</FieldLabel>
									<Textarea
										id="rotate-signing-key-pem"
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
						<Button type="submit" disabled={rotate.isPending || !signingKey}>
							{rotate.isPending ? "Rotating…" : "Rotate key"}
						</Button>
					</SheetFooter>
				</form>
			</SheetContent>
		</Sheet>
	);
}
