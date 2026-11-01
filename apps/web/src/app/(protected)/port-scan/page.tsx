import { Suspense } from "react";
import { PortScanClient } from "@/components/tools/port-scan-client";

export default function PortScanPage() {
  // Suspense: the tool reads ?target= from the URL.
  return (
    <Suspense>
      <PortScanClient />
    </Suspense>
  );
}
