import { AdminTitleEditor } from "@/components/admin/AdminTitleEditor";
import { notFound } from "next/navigation";
import { Suspense } from "react";

export default async function EditMoviePage({ params }: { params: Promise<{ id: string }> }) {
  const id = Number((await params).id);
  if (!Number.isInteger(id) || id < 1) notFound();
  return <Suspense fallback={<p className="animate-pulse">Preparing editor…</p>}><AdminTitleEditor kind="movies" titleId={id} /></Suspense>;
}
