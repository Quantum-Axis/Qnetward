import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "@/features/landing/landing.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Qnetward — Network, infrastructure, and security observability",
  description:
    "Probe, monitor, and protect every network and server from one platform. Live ping, port scan, DNS, alerting, and team workspaces.",
};

export default function MarketingLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <div className={`lp-root ${inter.variable} ${jetbrainsMono.variable}`}>{children}</div>;
}
