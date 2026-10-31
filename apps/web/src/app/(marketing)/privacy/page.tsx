import type { Metadata } from "next";
import { LegalPage, type LegalSection } from "@/features/landing/legal-page";

export const metadata: Metadata = {
  title: "Privacy Policy — Qnetward",
};

const sections: LegalSection[] = [
  {
    heading: "What we collect",
    body: [
      "Account details: your name, email address, a bcrypt hash of your password, and workspace memberships and roles.",
      "Service data: the targets you probe, the results, alert rules, and exports you create.",
      "Operational data: IP addresses, session records, and request logs used for security, rate limiting, and abuse investigation.",
    ],
  },
  {
    heading: "How we use it",
    body: [
      "To run the service: execute probes, store history, send alert and verification emails, and show your workspace to its members.",
      "To keep it safe: detect abuse, enforce rate limits, and respond to reports from network operators.",
      "We do not sell your personal data, and we do not use your probe results for advertising.",
    ],
  },
  {
    heading: "Where it is stored",
    body: [
      "Data is stored in PostgreSQL with TimescaleDB, Redis, and S3-compatible object storage, hosted with our infrastructure providers. Our primary region is Singapore.",
      "Transactional email is delivered through our email provider. These providers process data only on our behalf.",
    ],
  },
  {
    heading: "Retention",
    body: [
      "Probe results are kept for the retention period of your plan and then deleted automatically. Account data is kept while your account is active.",
      "When you delete your account, we remove your personal data within 30 days, except where we must keep records for legal or abuse-handling reasons.",
    ],
  },
  {
    heading: "Your choices",
    body: [
      "You can view and update your profile in the console, export your history, and ask us to delete your account and data.",
    ],
  },
  {
    heading: "Changes",
    body: ["We will post updates here and email account owners about material changes."],
  },
];

export default function PrivacyPage() {
  return (
    <LegalPage
      title="Privacy Policy"
      updated="31 October 2026"
      intro="This policy explains what Qnetward collects, why, and what control you have over it."
      sections={sections}
    />
  );
}
