import { Button } from "#/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import { cn } from "#/lib/utils";
import { loginFormSchema } from "#/schemas/login-form.schema";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "@tanstack/react-form";
import {
  createFileRoute,
  Link,
  redirect,
  useNavigate,
  useSearch,
} from "@tanstack/react-router";
import { loginMutationOptions } from "#/api/auth";
import { toast } from "sonner";

export const Route = createFileRoute("/auth/login")({
  component: RouteComponent,
});

function RouteComponent() {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });

  const loginMutation = useMutation({
    ...loginMutationOptions(),
  });

  const form = useForm({
    validators: {
      onChange: loginFormSchema,
    },
    defaultValues: {
      email: "",
      password: "",
    },
    onSubmit: async (values) => {
      loginMutation.mutate(
        {
          email: values.value.email,
          password: values.value.password,
        },
        {
          onSuccess: (user) => {
            toast.success(`Welcome back!`);
            navigate({
              to: (search as any).redirect || "/app/dashboard",
            });
          },
          onError: (error) => {
            toast.error(error.message || "Login failed");
          },
        },
      );
    },
  });

  return (
    <div className={cn("flex flex-col gap-6")}>
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-xl">Welcome back</CardTitle>
          <CardDescription>Login with your Google account</CardDescription>
        </CardHeader>
        <CardContent>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              form.handleSubmit(e);
            }}
          >
            <FieldGroup>
              <Field>
                <Button variant="outline" type="button">
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                    <path
                      d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"
                      fill="currentColor"
                    />
                  </svg>
                  Login with Google
                </Button>
              </Field>
              <FieldSeparator className="*:data-[slot=field-separator-content]:bg-card">
                Or continue with
              </FieldSeparator>
              <form.Field name="email">
                {(field) => (
                  <>
                    <Field id="email">
                      <FieldLabel htmlFor="email">Email</FieldLabel>
                      <Input
                        id="email"
                        type="email"
                        placeholder="m@example.com"
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                      />
                      {!field.state.meta.isValid && (
                        <FieldError errors={field.state.meta.errors} />
                      )}
                    </Field>
                  </>
                )}
              </form.Field>
              <form.Field name="password">
                {(field) => (
                  <>
                    <Field id="password">
                      <div className="flex items-center">
                        <FieldLabel htmlFor="password">Password</FieldLabel>
                        <Link
                          to="/auth/reset-password"
                          className="ml-auto text-sm underline-offset-4 hover:underline"
                        >
                          Forgot your password?
                        </Link>
                      </div>
                      <Input
                        id="password"
                        type="password"
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                      />
                      {!field.state.meta.isValid && (
                        <FieldError errors={field.state.meta.errors} />
                      )}
                    </Field>
                  </>
                )}
              </form.Field>

              <Field>
                <Button type="submit">Login</Button>
                <FieldDescription className="text-center">
                  Don&apos;t have an account?{" "}
                  <Link to="/auth/signup">Sign up</Link>
                </FieldDescription>
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
      <FieldDescription className="px-6 text-center">
        By clicking continue, you agree to our{" "}
        <Link to="/terms">Terms of Service</Link> and{" "}
        <Link to="/privacy">Privacy Policy</Link>.
      </FieldDescription>
    </div>
  );
}
