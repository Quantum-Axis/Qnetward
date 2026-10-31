import { Inter, JetBrains_Mono } from "next/font/google";
import "@/features/landing/landing.css";
import "@/features/auth/auth.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains",
  subsets: ["latin"],
});

export default function AuthLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <div className={`lp-root lp-auth ${inter.variable} ${jetbrainsMono.variable}`}>{children}</div>;
}
