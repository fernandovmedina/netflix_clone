import { AdminTitleEditor } from "@/components/admin/AdminTitleEditor";
import { Suspense } from "react";

export default function NewMoviePage() {
  return <Suspense fallback={<p className="animate-pulse">Preparing editor…</p>}><AdminTitleEditor kind="movies" /></Suspense>;
}
