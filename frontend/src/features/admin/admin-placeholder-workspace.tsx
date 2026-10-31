"use client";

import { Card } from "@/components/ui/card";

export function AdminPlaceholderWorkspace({
  title,
  eyebrow,
  description,
}: {
  title: string;
  eyebrow: string;
  description: string;
}) {
  return (
    <div className="grid gap-6">
      <section className="dashboard-hero-panel rounded-[1.7rem] p-5 md:p-6">
        <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">{eyebrow}</p>
        <h2 className="mt-1.5 font-space text-[1.9rem] font-bold leading-tight text-white">{title}</h2>
        <p className="mt-1 max-w-3xl text-sm text-slate-400">{description}</p>
      </section>

      <Card className="rounded-[1.5rem]">
        <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Planned Admin Surface</p>
        <p className="mt-2 text-lg font-semibold text-white">This route is now reserved under the admin RBAC gate.</p>
        <p className="mt-3 text-sm text-slate-400">
          The panel shell is live. Next implementation should fill this route with platform-level user, workspace, or system data.
        </p>
      </Card>
    </div>
  );
}
