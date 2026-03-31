"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { SessionResponse } from "@/lib/types";

type PreviewActivateResponse = {
  ok: boolean;
  message: string;
};

export function PreviewAccessWorkspace() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const [token, setToken] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const tokenFromQuery = useMemo(() => searchParams.get("token")?.trim() ?? "", [searchParams]);
  const effectiveToken = token || tokenFromQuery;

  const sessionQuery = useQuery({
    queryKey: ["session"],
    queryFn: () => apiRequest<SessionResponse>("/api/auth/session"),
  });

  const activatePreview = useMutation({
    mutationFn: (value: string) =>
      apiRequest<PreviewActivateResponse>("/api/auth/preview", {
        method: "POST",
        body: JSON.stringify({ token: value }),
      }),
    onSuccess: async () => {
      setFormError(null);
      await queryClient.invalidateQueries({ queryKey: ["session"] });
      router.replace("/");
    },
    onError: (error: ApiError) => {
      setFormError(error.message);
    },
  });

  useEffect(() => {
    if (sessionQuery.data?.authenticated) {
      router.replace("/");
    }
  }, [router, sessionQuery.data?.authenticated]);

  useEffect(() => {
    if (tokenFromQuery && !activatePreview.isPending && !sessionQuery.data?.authenticated) {
      activatePreview.mutate(tokenFromQuery);
    }
  }, [activatePreview, sessionQuery.data?.authenticated, tokenFromQuery]);

  return (
    <div className="dashboard-shell grid-shell flex min-h-screen items-center justify-center px-6 py-12">
      <div className="surface w-full max-w-lg rounded-3xl p-8">
        <p className="text-xs uppercase tracking-[0.24em] text-slate-400">NetWatcher Preview</p>
        <h1 className="mt-3 font-space text-3xl font-bold text-white">Contributor access</h1>
        <p className="mt-3 text-sm leading-7 text-slate-400">
          Enter the temporary preview token to open the dashboard in read-only mode. Mutating actions stay blocked.
        </p>

        <form
          className="mt-8 grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            setFormError(null);
            activatePreview.mutate(effectiveToken.trim());
          }}
        >
          <div>
            <label className="field-label" htmlFor="preview-token">Preview token</label>
            <Input
              id="preview-token"
              type="password"
              placeholder="Enter shared preview token"
              value={effectiveToken}
              onChange={(event) => setToken(event.target.value)}
            />
          </div>

          {formError ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{formError}</p> : null}

          <Button type="submit" disabled={activatePreview.isPending || effectiveToken.trim().length === 0}>
            {activatePreview.isPending ? "Granting preview..." : "Open read-only dashboard"}
          </Button>
        </form>
      </div>
    </div>
  );
}
