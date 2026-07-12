import { z } from "zod";

export const loginFormSchema = z.object({
  email: z.email("Email is required"),
  password: z.string().min(8, "Password is required"),
});