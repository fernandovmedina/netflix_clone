"use client";

import { AdminTitlePreview } from "@/components/admin/AdminTitlePreview";
import { StatusPill } from "@/components/admin/StatusPill";
import { adminApi, artworkUrl, type CatalogItem } from "@/utils/api/client";
import { Edit2, Eye, Trash2 } from "@deemlol/next-icons";
import Image from "next/image";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

type Row = {
  item: CatalogItem;
  key: number;
  kind: "movies" | "series";
  entityId: number | null;
  editHref: string;
  status: string;
};

function toRow(item: CatalogItem): Row {
  const kind = item.content_type.toLowerCase().includes("movie") ? "movies" : "series";
  const titleId = item.title_id ?? item.id;
  const entityId = (kind === "movies" ? item.movie_id : item.series_id) ?? null;
  // The movie editor is addressed by movie id; the series editor by title id.
  const routeId = kind === "movies" ? item.movie_id ?? titleId : titleId;
  return {
    item,
    key: titleId,
    kind,
    entityId,
    editHref: `/admin/${kind}/${routeId}`,
    status: item.asset_status ?? (item.asset_id ? "ready" : "draft"),
  };
}

export default function AdminDashboard() {
  const [rows, setRows] = useState<Row[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [preview, setPreview] = useState<Row | null>(null);
  const [confirming, setConfirming] = useState<Row | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    adminApi.titles()
      .then((titles) => { setRows(titles.map(toRow)); setError(""); })
      .catch((reason: unknown) => setError(reason instanceof Error ? reason.message : "Unable to load titles."))
      .finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const remove = async () => {
    if (!confirming?.entityId) return;
    setDeleting(true);
    setError("");
    try {
      await adminApi.deleteTitle(confirming.kind, confirming.entityId);
      setRows((current) => current.filter((row) => row.key !== confirming.key));
      setNotice(`“${confirming.item.title ?? `Title ${confirming.key}`}” was deleted.`);
      setConfirming(null);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to delete this title.");
    } finally {
      setDeleting(false);
    }
  };

  return <div>
    <div className="flex flex-wrap items-end justify-between gap-5"><div className="min-w-0"><p className="font-bold uppercase tracking-[0.25em] text-red-500">Content operations</p><h1 className="mt-2 text-4xl font-black sm:text-5xl">Title dashboard</h1><p className="mt-3 text-zinc-400">Published and draft catalog records, including titles still waiting for media.</p></div><div className="grid w-full grid-cols-2 gap-3 sm:flex sm:w-auto"><Link href="/admin/movies/new" className="whitespace-nowrap rounded bg-white px-4 py-3 text-center font-bold text-black">New movie</Link><Link href="/admin/series/new" className="whitespace-nowrap rounded bg-red-600 px-4 py-3 text-center font-bold">New series</Link></div></div>
    {notice && <p className="mt-6 rounded bg-emerald-950/60 p-4 text-emerald-200">{notice}</p>}
    {loading && <p className="mt-16 animate-pulse text-center text-zinc-300">Loading the complete catalog…</p>}
    {error && <p className="mt-10 rounded bg-red-950/60 p-5 text-red-200">{error}</p>}
    {!loading && !error && rows.length === 0 && <p className="mt-16 text-center text-zinc-400">No titles exist yet. Create the first one above.</p>}
    {!loading && !error && rows.length > 0 && <div className="mt-9 max-w-full overflow-x-auto rounded-xl border border-zinc-800">
      <div className="hidden grid-cols-[5rem_1fr_9rem_8rem_9rem] gap-4 bg-zinc-900 px-5 py-3 text-xs font-bold uppercase tracking-wider text-zinc-400 md:grid"><span>Artwork</span><span>Title</span><span>Type</span><span>Status</span><span className="text-right">Actions</span></div>
      <div className="divide-y divide-zinc-800">{rows.map((row) => <article key={row.key} className="grid min-w-0 grid-cols-[4rem_minmax(0,1fr)_auto] items-center gap-3 bg-zinc-950 px-4 py-4 hover:bg-zinc-900 sm:gap-4 md:grid-cols-[5rem_minmax(0,1fr)_9rem_8rem_9rem] md:px-5">
        <div className="relative aspect-video overflow-hidden rounded bg-zinc-800">{row.item.thumbnail_url && <Image src={artworkUrl(row.item.thumbnail_url)} alt="" fill sizes="5rem" className="object-cover" unoptimized />}</div>
        <div className="min-w-0"><h2 className="truncate font-bold">{row.item.title ?? `Title ${row.key}`}</h2><p className="mt-1 truncate text-xs text-zinc-500">#{row.key} · {row.item.published ? "Published" : "Unpublished"}</p></div>
        <div className="md:hidden"><StatusPill status={row.status} /></div>
        <span className="hidden text-sm text-zinc-300 md:block">{row.item.content_type}</span>
        <div className="hidden md:block"><StatusPill status={row.status} /></div>
        <div className="col-span-3 flex items-center justify-start gap-2 md:col-span-1 md:justify-end">
          <RowAction label={`View ${row.item.title ?? "title"}`} onClick={() => setPreview(row)}><Eye size={18} /></RowAction>
          <RowAction label={`Edit ${row.item.title ?? "title"}`} href={row.editHref}><Edit2 size={18} /></RowAction>
          <RowAction label={row.entityId ? `Delete ${row.item.title ?? "title"}` : "Delete is unavailable for this record"} destructive disabled={!row.entityId} onClick={() => setConfirming(row)}><Trash2 size={18} /></RowAction>
        </div>
      </article>)}</div>
    </div>}

    {preview && <AdminTitlePreview item={preview.item} editHref={preview.editHref} onClose={() => setPreview(null)} />}

    {confirming && <div role="dialog" aria-modal="true" aria-label="Confirm deletion" className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4" onClick={() => !deleting && setConfirming(null)}>
      <div className="w-full max-w-md rounded-xl border border-zinc-700 bg-zinc-950 p-6" onClick={(event) => event.stopPropagation()}>
        <h2 className="text-2xl font-black">Delete this title?</h2>
        <p className="mt-3 text-zinc-300">“{confirming.item.title ?? `Title ${confirming.key}`}” will be removed from the catalog and will stop being served to viewers.</p>
        <div className="mt-6 flex flex-col gap-3 sm:flex-row-reverse">
          <button type="button" onClick={remove} disabled={deleting} className="min-h-11 flex-1 rounded bg-red-600 px-5 font-bold hover:bg-red-700 disabled:cursor-wait disabled:opacity-60">{deleting ? "Deleting…" : "Delete"}</button>
          <button type="button" onClick={() => setConfirming(null)} disabled={deleting} className="min-h-11 rounded border border-zinc-600 px-5 font-bold hover:border-white disabled:opacity-60">Cancel</button>
        </div>
      </div>
    </div>}
  </div>;
}

type RowActionProps = {
  label: string;
  children: React.ReactNode;
  href?: string;
  onClick?: () => void;
  destructive?: boolean;
  disabled?: boolean;
};

function RowAction({ label, children, href, onClick, destructive, disabled }: RowActionProps) {
  const className = `flex h-11 w-11 shrink-0 items-center justify-center rounded border ${destructive ? "border-red-800 text-red-300 hover:border-red-500 hover:bg-red-950/60" : "border-zinc-700 text-zinc-200 hover:border-white hover:bg-zinc-800"} disabled:cursor-not-allowed disabled:opacity-40`;
  if (href) return <Link href={href} aria-label={label} title={label} className={className}>{children}</Link>;
  return <button type="button" aria-label={label} title={label} disabled={disabled} onClick={onClick} className={className}>{children}</button>;
}
