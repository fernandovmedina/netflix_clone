import { AdminTitleEditor } from "@/components/admin/AdminTitleEditor";
import { pageMetadata } from "@/utils/metadata";
import { Suspense } from "react";

export const metadata = pageMetadata("Add Movie", "Create a movie and upload its artwork and video.", { authenticated: true });

export default function NewMoviePage() {
  return <Suspense fallback={<p className="animate-pulse">Preparing editor…</p>}><AdminTitleEditor kind="movies" /></Suspense>;
}
