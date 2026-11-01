import { Suspense } from "react";
import { PingToolClient } from "@/components/tools/ping-tool-client";

export default function TCPPingPage() {
  // Suspense: the tool reads ?target= from the URL.
  return (
    <Suspense>
      <PingToolClient mode="tcpping" />
    </Suspense>
  );
}
