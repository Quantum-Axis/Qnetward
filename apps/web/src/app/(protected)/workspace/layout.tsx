import { ReactNode } from "react";
import { SettingsShell } from "@/components/settings/settings-shell";

export default function WorkspaceSettingsLayout({ children }: { children: ReactNode }) {
  return <SettingsShell context="workspace">{children}</SettingsShell>;
}
