import { LogIn } from "lucide-react";
import { AuthLayoutCard } from "@/components/auth/auth-layout-card";
import { GuestOnly } from "@/components/auth/guest-only";
import { LoginForm } from "@/components/forms/login-form";

export default function LoginPage() {
  return (
    <GuestOnly>
      <AuthLayoutCard
        headerAction={{ prompt: "New to Qnetward?", label: "Create an account", href: "/signup" }}
        icon={<LogIn />}
        title="Sign in to Qnetward"
        subtitle="Welcome back. Pick up where your team left off."
      >
        <LoginForm />
      </AuthLayoutCard>
    </GuestOnly>
  );
}
