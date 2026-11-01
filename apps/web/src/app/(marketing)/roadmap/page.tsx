import type { Metadata } from "next";
import { RoadmapPage } from "@/features/landing/roadmap-page";

export const metadata: Metadata = {
  title: "Roadmap — Qnetward",
  description: "What Qnetward does today, what's next, and what comes after.",
};

export default function Roadmap() {
  return <RoadmapPage />;
}
