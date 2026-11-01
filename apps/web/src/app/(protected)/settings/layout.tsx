import { ReactNode } from "react";
import { SettingsShell } from "@/components/settings/settings-shell";

export default function PersonalSettingsLayout({ children }: { children: ReactNode }) {
  return <SettingsShell context="personal">{children}</SettingsShell>;
}
