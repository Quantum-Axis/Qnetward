"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { LockKeyhole, Mail } from "lucide-react";
import { AuthInputField } from "@/components/auth/auth-input-field";
import { apiRequest, ApiError } from "@/lib/api";
import { SessionResponse } from "@/lib/types";

const schema = z.object({
  email: z.email("Enter a valid email address"),
  password: z.string().min(1, "Enter your password"),
});

type FormValues = z.infer<typeof schema>;

export function LoginForm() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      email: "",
      password: "",
    },
  });

  const login = useMutation({
    mutationFn: (values: FormValues) => apiRequest<SessionResponse>("/api/auth/login", { method: "POST", body: JSON.stringify(values) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["session"] });
      router.replace("/overview");
    },
    onError: (error: ApiError) => {
      form.setError("root", { message: error.message });
    },
  });

  const demoLogin = useMutation({
    mutationFn: () => apiRequest<SessionResponse>("/api/auth/demo-login", { method: "POST", body: JSON.stringify({}) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["session"] });
      router.replace("/overview");
    },
    onError: (error: ApiError) => {
      form.setError("root", { message: error.message });
    },
  });

  return (
    <form className="grid gap-3.5" noValidate onSubmit={form.handleSubmit((values) => login.mutate(values))}>
      <AuthInputField
        id="login-email"
        label="Email"
        icon={Mail}
        error={form.formState.errors.email?.message}
        inputProps={{
          type: "email",
          autoComplete: "email",
          autoFocus: true,
          ...form.register("email"),
        }}
      />
      <AuthInputField
        id="login-password"
        label="Password"
        icon={LockKeyhole}
        passwordToggle
        labelAction={
          <Link className="auth-inline-link" href="/forgot-password">
            Forgot password?
          </Link>
        }
        error={form.formState.errors.password?.message}
        inputProps={{
          autoComplete: "current-password",
          ...form.register("password"),
        }}
      />
      {form.formState.errors.root && (
        <p className="auth-notice-error" role="alert">
          {form.formState.errors.root.message}
        </p>
      )}
      <button className="auth-primary-button mt-1" type="submit" disabled={login.isPending}>
        {login.isPending ? "Signing in…" : "Sign in"}
      </button>

      <div className="auth-divider">Or continue with</div>

      <button className="auth-muted-button" type="button" disabled={demoLogin.isPending} onClick={() => demoLogin.mutate()}>
        {demoLogin.isPending ? "Opening demo…" : "Explore the demo workspace"}
      </button>
      <p className="-mt-1 text-center text-[12px] text-[var(--lp-subtle)]">Shared sandbox. Other visitors can see what you run.</p>
    </form>
  );
}
