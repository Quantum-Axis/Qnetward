import { Suspense } from "react";
import { PreviewAccessWorkspace } from "@/features/preview-access/preview-access-workspace";

export default function PreviewPage() {
  return (
    <Suspense fallback={null}>
      <PreviewAccessWorkspace />
    </Suspense>
  );
}
