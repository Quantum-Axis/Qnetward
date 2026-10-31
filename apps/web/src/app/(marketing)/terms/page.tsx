import type { Metadata } from "next";
import { LegalPage, type LegalSection } from "@/features/landing/legal-page";

export const metadata: Metadata = {
  title: "Terms of Service — Qnetward",
};

const sections: LegalSection[] = [
  {
    heading: "Using the service",
    body: [
      "You must be at least 18 and able to enter a binding agreement to create an account. If you use Qnetward on behalf of an organization, you confirm you are authorized to accept these terms for it.",
      "You are responsible for the activity in your workspaces, for keeping your credentials safe, and for the people you invite.",
    ],
  },
  {
    heading: "Acceptable use and network testing",
    body: [
      "Qnetward sends network traffic, such as ICMP echo requests, TCP connections, and DNS queries, to targets you choose. You may only direct it at systems you own or are explicitly authorized to test.",
      "You must not use Qnetward to disrupt, overload, or gain unauthorized access to any system; to scan networks without permission; or to evade the rate limits and safeguards built into the service.",
      "Deeper security scanning, when available, will require proof that you own the target and may require a signed authorization. We may refuse or stop any scan at our discretion.",
    ],
  },
  {
    heading: "Abuse and enforcement",
    body: [
      "We log probe activity, including the account, target, and time, to investigate abuse reports and protect third parties.",
      "We may suspend or close accounts that break these terms, and we may share relevant records with network operators or authorities when required by law or to stop ongoing harm.",
    ],
  },
  {
    heading: "Beta service",
    body: [
      "Qnetward is in beta. Features may change or be removed, and the service is provided without uptime guarantees during this period.",
      "Free use during the beta does not commit you to a paid plan. We will announce pricing before paid plans take effect.",
    ],
  },
  {
    heading: "Your data",
    body: [
      "You own the data you put into Qnetward, including targets, results, and alert rules. You grant us the rights needed to store, process, and display it to operate the service.",
      "Our Privacy Policy explains what we collect and how we use it.",
    ],
  },
  {
    heading: "Liability",
    body: [
      "To the extent permitted by law, Qnetward is provided “as is”, and we are not liable for indirect or consequential losses, or for results you act on from the service.",
    ],
  },
  {
    heading: "Changes",
    body: ["We may update these terms. If a change is material, we will notify account owners by email before it takes effect."],
  },
];

export default function TermsPage() {
  return (
    <LegalPage
      title="Terms of Service"
      updated="31 October 2026"
      intro="These terms govern your use of Qnetward's website, console, API, and command-line tools. By creating an account or using the service, you agree to them."
      sections={sections}
    />
  );
}
