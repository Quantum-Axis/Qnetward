import { ReactNode } from "react";
import { Inter, JetBrains_Mono } from "next/font/google";
import { AuthGuard } from "@/components/app/auth-guard";
import "@/features/landing/landing.css";
import "@/features/console/console.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains",
  subsets: ["latin"],
});

export default function ProtectedLayout({ children }: { children: ReactNode }) {
  return (
    <div className={`lp-root lp-console ${inter.variable} ${jetbrainsMono.variable}`}>
      <AuthGuard>{children}</AuthGuard>
    </div>
  );
}
