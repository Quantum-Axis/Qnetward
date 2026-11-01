import { Suspense } from "react";
import { PingToolClient } from "@/components/tools/ping-tool-client";

export default function ICMPPingPage() {
  // Suspense: the tool reads ?target= from the URL.
  return (
    <Suspense>
      <PingToolClient mode="ping" />
    </Suspense>
  );
}
