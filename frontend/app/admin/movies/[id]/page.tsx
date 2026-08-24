import { AdminTitleEditor } from "@/components/admin/AdminTitleEditor";
import { getAdminTitle } from "@/utils/admin-title";
import { pageMetadata } from "@/utils/metadata";
import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

type Props = { params: Promise<{ id: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const id = Number((await params).id);
  const detail = await getAdminTitle("movies", id);
  return pageMetadata(detail?.title ? `Edit ${detail.title}` : "Edit Movie", "Edit movie metadata, artwork, and video.", { authenticated: true });
}

export default async function EditMoviePage({ params }: Props) {
  const id = Number((await params).id);
  if (!Number.isInteger(id) || id < 1) notFound();
  const detail = await getAdminTitle("movies", id);
  return <Suspense fallback={<p className="animate-pulse">Preparing editor…</p>}><AdminTitleEditor kind="movies" titleId={id} initialDetail={detail ?? undefined} /></Suspense>;
}
