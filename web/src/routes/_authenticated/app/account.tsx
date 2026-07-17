import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
} from "#/components/ui/card";
import {
  Combobox,
  ComboboxEmpty,
  ComboboxContent,
  ComboboxInput,
  ComboboxList,
  ComboboxItem,
} from "#/components/ui/combobox";
import {
  FieldGroup,
  Field,
  FieldSeparator,
  FieldLabel,
  FieldError,
  FieldDescription,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import { Separator } from "#/components/ui/separator";
import { useForm } from "@tanstack/react-form";
import { createFileRoute, Link } from "@tanstack/react-router";
import { InfoIcon, Upload } from "lucide-react";

export const Route = createFileRoute("/_authenticated/app/account")({
  component: RouteComponent,
});

const timezones = [
  { label: "(UTC-12:00) International Date Line West", value: "Etc/GMT+12" },
  { label: "(UTC-11:00) American Samoa", value: "Pacific/Pago_Pago" },
  { label: "(UTC-10:00) Hawaii", value: "Pacific/Honolulu" },
  { label: "(UTC-09:00) Alaska", value: "America/Anchorage" },
  {
    label: "(UTC-08:00) Pacific Time (US & Canada)",
    value: "America/Los_Angeles",
  },
  { label: "(UTC-07:00) Mountain Time (US & Canada)", value: "America/Denver" },
  { label: "(UTC-07:00) Arizona", value: "America/Phoenix" },
  { label: "(UTC-06:00) Central Time (US & Canada)", value: "America/Chicago" },
  {
    label: "(UTC-05:00) Eastern Time (US & Canada)",
    value: "America/New_York",
  },
  { label: "(UTC-05:00) Bogotá, Lima", value: "America/Bogota" },
  { label: "(UTC-04:00) Atlantic Time (Canada)", value: "America/Halifax" },
  { label: "(UTC-04:00) Santiago", value: "America/Santiago" },
  {
    label: "(UTC-03:00) Buenos Aires",
    value: "America/Argentina/Buenos_Aires",
  },
  { label: "(UTC-03:00) São Paulo", value: "America/Sao_Paulo" },
  { label: "(UTC-02:00) Mid-Atlantic", value: "Etc/GMT+2" },
  { label: "(UTC-01:00) Azores", value: "Atlantic/Azores" },
  { label: "(UTC±00:00) UTC", value: "UTC" },
  { label: "(UTC±00:00) London", value: "Europe/London" },
  { label: "(UTC+01:00) Berlin", value: "Europe/Berlin" },
  { label: "(UTC+01:00) Paris", value: "Europe/Paris" },
  { label: "(UTC+01:00) Madrid", value: "Europe/Madrid" },
  { label: "(UTC+01:00) Rome", value: "Europe/Rome" },
  { label: "(UTC+02:00) Athens", value: "Europe/Athens" },
  { label: "(UTC+02:00) Cairo", value: "Africa/Cairo" },
  { label: "(UTC+02:00) Helsinki", value: "Europe/Helsinki" },
  { label: "(UTC+03:00) Istanbul", value: "Europe/Istanbul" },
  { label: "(UTC+03:00) Moscow", value: "Europe/Moscow" },
  { label: "(UTC+03:00) Riyadh", value: "Asia/Riyadh" },
  { label: "(UTC+03:30) Tehran", value: "Asia/Tehran" },
  { label: "(UTC+04:00) Dubai", value: "Asia/Dubai" },
  { label: "(UTC+04:00) Baku", value: "Asia/Baku" },
  { label: "(UTC+04:30) Kabul", value: "Asia/Kabul" },
  { label: "(UTC+05:00) Karachi", value: "Asia/Karachi" },
  { label: "(UTC+05:30) India Standard Time", value: "Asia/Kolkata" },
  { label: "(UTC+05:45) Kathmandu", value: "Asia/Kathmandu" },
  { label: "(UTC+06:00) Dhaka", value: "Asia/Dhaka" },
  { label: "(UTC+06:30) Yangon", value: "Asia/Yangon" },
  { label: "(UTC+07:00) Bangkok", value: "Asia/Bangkok" },
  { label: "(UTC+07:00) Jakarta", value: "Asia/Jakarta" },
  { label: "(UTC+08:00) Singapore", value: "Asia/Singapore" },
  { label: "(UTC+08:00) Hong Kong", value: "Asia/Hong_Kong" },
  { label: "(UTC+08:00) Beijing", value: "Asia/Shanghai" },
  { label: "(UTC+08:00) Perth", value: "Australia/Perth" },
  { label: "(UTC+09:00) Tokyo", value: "Asia/Tokyo" },
  { label: "(UTC+09:00) Seoul", value: "Asia/Seoul" },
  { label: "(UTC+09:30) Adelaide", value: "Australia/Adelaide" },
  { label: "(UTC+09:30) Darwin", value: "Australia/Darwin" },
  { label: "(UTC+10:00) Sydney", value: "Australia/Sydney" },
  { label: "(UTC+10:00) Melbourne", value: "Australia/Melbourne" },
  { label: "(UTC+10:00) Brisbane", value: "Australia/Brisbane" },
  { label: "(UTC+11:00) Solomon Islands", value: "Pacific/Guadalcanal" },
  { label: "(UTC+12:00) Auckland", value: "Pacific/Auckland" },
  { label: "(UTC+12:00) Fiji", value: "Pacific/Fiji" },
  { label: "(UTC+13:00) Samoa", value: "Pacific/Apia" },
];

function RouteComponent() {
  const form = useForm({
    defaultValues: {
      name: "",
      email: "",
      password: "",
      timezone: "",
    },
    onSubmit: async (values) => {},
  });

  return (
    <div className="w-full h-full flex justify-center items-center">
      <Card className="w-160">
        <CardHeader className="font-bold text-xl">Your profile</CardHeader>
        <CardContent>
          <div className="flex gap-4 border-t">
            <div className="border-r p-4 flex flex-col gap-4">
              <Avatar className="rounded-md size-30">
                <AvatarImage alt={"Fatih"} />
                <AvatarFallback>FC</AvatarFallback>
              </Avatar>

              <div className="avatar-actions flex gap-2">
                <Button>
                  <Upload />
                  Choose new
                </Button>
                <Button variant={"outline"}>Remove</Button>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">
                  Accepted formats: PNG, JPG, JPEG. Max size: 10MB.
                </p>
              </div>
            </div>
            <div className="user-profile-settings w-full pt-4">
              <form
              // onSubmit={(e) => {
              //   e.preventDefault();
              //   form.handleSubmit(e);
              // }}
              >
                <FieldGroup>
                  <form.Field name="name">
                    {(field) => (
                      <>
                        <Field id="name">
                          <FieldLabel htmlFor="name">Full name</FieldLabel>
                          <Input
                            id="name"
                            type="text"
                            placeholder="Fatih Cure"
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
                  <form.Field name="email">
                    {(field) => (
                      <>
                        <Field id="email">
                          <FieldLabel htmlFor="email">Email</FieldLabel>
                          <Input
                            disabled
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
                  <FieldSeparator className="*:data-[slot=field-separator-content]:bg-card"></FieldSeparator>
                  <form.Field name="timezone">
                    {(field) => (
                      <>
                        <Field id="email">
                          <FieldLabel htmlFor="email">Timezone</FieldLabel>
                          <Combobox items={timezones} autoHighlight>
                            <ComboboxInput
                              placeholder="Select a timezone"
                              showClear
                            />
                            <ComboboxContent>
                              <ComboboxEmpty>No items found.</ComboboxEmpty>
                              <ComboboxList>
                                {(item) => (
                                  <ComboboxItem key={item} value={item.value}>
                                    {item.label}
                                  </ComboboxItem>
                                )}
                              </ComboboxList>
                            </ComboboxContent>
                          </Combobox>
                          {!field.state.meta.isValid && (
                            <FieldError errors={field.state.meta.errors} />
                          )}
                        </Field>
                      </>
                    )}
                  </form.Field>
                </FieldGroup>
              </form>
            </div>
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button variant={"default"}>Save</Button>
        </CardFooter>
      </Card>
    </div>
  );
}
