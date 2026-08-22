"use client";

import { StatusPill } from "@/components/admin/StatusPill";
import { adminApi, artworkUrl, type CatalogItem } from "@/utils/api/client";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useState } from "react";

export default function AdminDashboard() {
  const [titles, setTitles] = useState<CatalogItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    adminApi.titles().then(setTitles).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : "Unable to load titles.")).finally(() => setLoading(false));
  }, []);

  return <div>
    <div className="flex flex-wrap items-end justify-between gap-5"><div><p className="font-bold uppercase tracking-[0.25em] text-red-500">Content operations</p><h1 className="mt-2 text-4xl font-black sm:text-5xl">Title dashboard</h1><p className="mt-3 text-zinc-400">Published and draft catalog records, including titles still waiting for media.</p></div><div className="flex gap-3"><Link href="/admin/movies/new" className="rounded bg-white px-4 py-3 font-bold text-black">New movie</Link><Link href="/admin/series/new" className="rounded bg-red-600 px-4 py-3 font-bold">New series</Link></div></div>
    {loading && <p className="mt-16 animate-pulse text-center text-zinc-300">Loading the complete catalog…</p>}
    {error && <p className="mt-10 rounded bg-red-950/60 p-5 text-red-200">{error}</p>}
    {!loading && !error && titles.length === 0 && <p className="mt-16 text-center text-zinc-400">No titles exist yet. Create the first one above.</p>}
    {!loading && !error && titles.length > 0 && <div className="mt-9 max-w-full overflow-x-auto rounded-xl border border-zinc-800">
      <div className="hidden grid-cols-[5rem_1fr_9rem_8rem_5rem] gap-4 bg-zinc-900 px-5 py-3 text-xs font-bold uppercase tracking-wider text-zinc-400 md:grid"><span>Artwork</span><span>Title</span><span>Type</span><span>Status</span><span /></div>
      <div className="divide-y divide-zinc-800">{titles.map((item) => {
        const kind = item.content_type.toLowerCase().includes("movie") ? "movies" : "series";
        const status = item.asset_id ? "ready" : "draft";
        return <article key={item.title_id ?? item.id} className="grid grid-cols-[4rem_1fr_auto] items-center gap-4 bg-zinc-950 px-4 py-4 hover:bg-zinc-900 md:grid-cols-[5rem_1fr_9rem_8rem_5rem] md:px-5">
          <div className="relative aspect-video overflow-hidden rounded bg-zinc-800">{item.thumbnail_url && <Image src={artworkUrl(item.thumbnail_url)} alt="" fill className="object-cover" unoptimized />}</div>
          <div className="min-w-0"><h2 className="truncate font-bold">{item.title ?? `Title ${item.id}`}</h2><p className="mt-1 truncate text-xs text-zinc-500">#{item.title_id ?? item.id} · {item.published ? "Published" : "Unpublished"}</p></div>
          <div className="md:hidden"><StatusPill status={status} /></div>
          <span className="hidden text-sm text-zinc-300 md:block">{item.content_type}</span>
          <div className="hidden md:block"><StatusPill status={status} /></div>
          <Link href={`/admin/${kind}/${item.title_id ?? item.id}`} className="hidden rounded border border-zinc-600 px-3 py-2 text-center text-sm font-bold hover:border-white md:block">Edit</Link>
          <Link href={`/admin/${kind}/${item.title_id ?? item.id}`} className="col-span-3 rounded border border-zinc-700 py-2 text-center text-sm md:hidden">Edit title</Link>
        </article>;
      })}</div>
    </div>}
  </div>;
}
