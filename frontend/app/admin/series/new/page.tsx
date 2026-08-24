import { AdminTitleEditor } from "@/components/admin/AdminTitleEditor";
import { pageMetadata } from "@/utils/metadata";
import { Suspense } from "react";

export const metadata = pageMetadata("Add Series", "Create a series, seasons, episodes, and media.", { authenticated: true });

export default function NewSeriesPage() {
  return <Suspense fallback={<p className="animate-pulse">Preparing editor…</p>}><AdminTitleEditor kind="series" /></Suspense>;
}
