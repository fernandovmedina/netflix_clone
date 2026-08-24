"use client";

import { StatusPill } from "@/components/admin/StatusPill";
import { artworkUrl, catalogApi, type CatalogItem, type TitleDetail } from "@/utils/api/client";
import { X } from "@deemlol/next-icons";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useState } from "react";

type AdminTitlePreviewProps = {
  item: CatalogItem;
  editHref: string;
  onClose: () => void;
};

function durationLabel(duration?: number | string): string {
  if (typeof duration === "string") return duration;
  if (typeof duration === "number" && duration > 0) return `${duration} min`;
  return "";
}

export function AdminTitlePreview({ item, editHref, onClose }: AdminTitlePreviewProps) {
  const [detail, setDetail] = useState<TitleDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    catalogApi
      .detail(item)
      .then((value) => active && setDetail(value))
      .catch((reason: unknown) => {
        if (active) setError(reason instanceof Error ? reason.message : "Unable to load this title.");
      })
      .finally(() => active && setLoading(false));
    return () => { active = false; };
  }, [item]);

  useEffect(() => {
    const close = (event: KeyboardEvent) => { if (event.key === "Escape") onClose(); };
    window.addEventListener("keydown", close);
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      window.removeEventListener("keydown", close);
      document.body.style.overflow = previous;
    };
  }, [onClose]);

  const shown = detail ?? item;
  const people = detail?.actors ?? detail?.cast ?? [];
  const tags = detail?.genres ?? detail?.categories ?? [];
  const seasons = detail?.seasons ?? [];
  const episodeCount = seasons.reduce((total, season) => total + season.episodes.length, 0);

  return <div role="dialog" aria-modal="true" aria-label={`Preview of ${shown.title ?? "title"}`} className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/80 p-4 sm:p-8" onClick={onClose}>
    <div className="w-full max-w-3xl rounded-xl border border-zinc-700 bg-zinc-950 shadow-2xl" onClick={(event) => event.stopPropagation()}>
      <div className="relative aspect-video w-full overflow-hidden rounded-t-xl bg-zinc-900">
        {shown.thumbnail_url && <Image src={artworkUrl(shown.thumbnail_url)} alt="" fill sizes="(max-width: 768px) 100vw, 48rem" className="object-cover" unoptimized />}
        <div className="absolute inset-0 bg-linear-to-t from-zinc-950 via-zinc-950/30 to-transparent" />
        <button type="button" onClick={onClose} aria-label="Close preview" className="absolute right-3 top-3 flex h-10 w-10 items-center justify-center rounded-full bg-black/70 hover:bg-black"><X size={20} /></button>
      </div>
      <div className="p-5 sm:p-7">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="min-w-0 text-2xl font-black sm:text-3xl">{shown.title ?? `Title ${shown.id}`}</h2>
          <StatusPill status={shown.asset_status ?? (shown.asset_id ? "ready" : "draft")} />
          <StatusPill status={shown.published ? "published" : "draft"} />
        </div>
        <p className="mt-2 text-sm text-zinc-400">
          #{shown.title_id ?? shown.id} · {shown.content_type}
          {shown.year_released ? ` · ${shown.year_released}` : ""}
          {durationLabel(detail?.duration) ? ` · ${durationLabel(detail?.duration)}` : ""}
          {seasons.length > 0 ? ` · ${seasons.length} season${seasons.length === 1 ? "" : "s"}, ${episodeCount} episode${episodeCount === 1 ? "" : "s"}` : ""}
        </p>
        {loading && <p className="mt-5 animate-pulse text-zinc-400">Loading the full record…</p>}
        {error && <p className="mt-5 rounded bg-red-950/60 p-4 text-red-200">{error}</p>}
        {shown.description && <p className="mt-5 whitespace-pre-line text-zinc-200">{shown.description}</p>}
        <dl className="mt-5 grid gap-3 text-sm sm:grid-cols-2">
          {detail?.director && <div><dt className="text-zinc-500">Director</dt><dd className="mt-1">{detail.director}</dd></div>}
          {tags.length > 0 && <div><dt className="text-zinc-500">Genres</dt><dd className="mt-1">{tags.join(" · ")}</dd></div>}
          {people.length > 0 && <div className="sm:col-span-2"><dt className="text-zinc-500">Cast</dt><dd className="mt-1">{people.join(" · ")}</dd></div>}
        </dl>
        {seasons.length > 0 && <div className="mt-6 grid gap-4">
          {seasons.map((season, index) => <section key={season.season_id ?? season.id ?? index} className="rounded-lg border border-zinc-800 p-4">
            <h3 className="font-bold">Season {season.season_number ?? season.number ?? index + 1}</h3>
            <ul className="mt-3 grid gap-2">{season.episodes.map((episode, episodeIndex) => <li key={episode.episode_id ?? episode.id ?? episodeIndex} className="flex flex-wrap items-center gap-3 text-sm">
              <span className="min-w-0 flex-1 truncate">{episode.episode_number ?? episode.episode ?? episodeIndex + 1}. {episode.title}</span>
              <StatusPill status={episode.asset_status ?? (episode.asset_id ? "ready" : "draft")} />
            </li>)}
            {season.episodes.length === 0 && <li className="text-sm text-zinc-500">No episodes yet.</li>}</ul>
          </section>)}
        </div>}
        <div className="mt-7 flex flex-col gap-3 sm:flex-row">
          <Link href={editHref} className="flex min-h-11 flex-1 items-center justify-center rounded bg-white px-5 font-bold text-black">Edit this title</Link>
          <button type="button" onClick={onClose} className="min-h-11 rounded border border-zinc-600 px-5 font-bold hover:border-white">Close</button>
        </div>
      </div>
    </div>
  </div>;
}
