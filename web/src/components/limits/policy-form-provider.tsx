import {
  createContext,
  useContext,
  type PropsWithChildren,
} from "react";
import {
  useForm,
  type AnyFormApi,
} from "@tanstack/react-form";

export type PolicyFormValues = {
  name: string;

  enabled: boolean;

  scope:
    | "organization"
    | "team"
    | "project"
    | "user";

  targetId: string;

  priority: number;

  providers: string[];

  models: string[];

  modalities: string[];

  conditions: unknown[];

  limits: unknown[];

  action:
    | "reject"
    | "queue"
    | "fallback";

  fallbackModel: string;
};

const PolicyFormContext =
  createContext<AnyFormApi | null>(null);

export function PolicyFormProvider({
  children,
}: PropsWithChildren) {
  const form = useForm({
    defaultValues: {
      name: "",

      enabled: true,

      scope: "organization",

      targetId: "",

      priority: 100,

      providers: [],

      models: [],

      modalities: ["chat"],

      conditions: [],

      limits: [],

      action: "reject",

      fallbackModel: "",
    } satisfies PolicyFormValues,

    onSubmit: async ({ value }) => {
      console.log(value);
    },
  });

  return (
    <PolicyFormContext.Provider value={form}>
      {children}
    </PolicyFormContext.Provider>
  );
}

export function usePolicyForm() {
  const form = useContext(PolicyFormContext);

  if (!form) {
    throw new Error(
      "usePolicyForm must be used inside PolicyFormProvider",
    );
  }

  return form;
}