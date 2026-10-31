import { UserPlus } from "lucide-react";
import { AuthLayoutCard } from "@/components/auth/auth-layout-card";
import { GuestOnly } from "@/components/auth/guest-only";
import { SignupForm } from "@/components/forms/signup-form";

export default function SignupPage() {
  return (
    <GuestOnly>
      <AuthLayoutCard
        headerAction={{ prompt: "Already have an account?", label: "Sign in", href: "/login" }}
        icon={<UserPlus />}
        title="Create your account"
        subtitle="Takes about a minute. You can invite your team afterwards."
        wide
      >
        <SignupForm />
      </AuthLayoutCard>
    </GuestOnly>
  );
}
