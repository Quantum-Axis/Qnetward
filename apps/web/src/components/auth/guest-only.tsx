"use client";

import { QnetwardMark } from "@/features/landing/qnetward-mark";
import { ReactNode, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { apiRequest } from "@/lib/api";
import { SessionResponse } from "@/lib/types";

export function GuestOnly({ children }: { children: ReactNode }) {
  const router = useRouter();
  const session = useQuery({
    queryKey: ["session"],
    queryFn: () => apiRequest<SessionResponse>("/api/auth/session"),
  });

  useEffect(() => {
    if (!session.isLoading && session.data?.authenticated) {
      router.replace("/overview");
    }
  }, [router, session.data?.authenticated, session.isLoading]);

  if (session.isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center" role="status" aria-label="Checking your session">
        <span className="lp-pulse">
          <QnetwardMark size={32} />
        </span>
      </div>
    );
  }

  if (session.data?.authenticated) {
    return null;
  }

  return <>{children}</>;
}
