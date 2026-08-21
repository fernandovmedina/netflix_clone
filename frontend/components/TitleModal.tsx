"use client";

import {
  artworkUrl,
  catalogApi,
  isPlayable,
  titleId,
  userApi,
  watchHref,
  type CatalogItem,
  type TitleDetail,
} from "@/utils/api/client";
import { Play, Plus } from "@deemlol/next-icons";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

type TitleModalProps = {
  item: CatalogItem;
  onClose: () => void;
};

function durationLabel(duration?: number | string): string {
  if (typeof duration === "string") return duration;
  if (typeof duration === "number") return `${duration} min`;
  return "";
}

export function TitleModal({ item, onClose }: TitleModalProps) {
  const [detail, setDetail] = useState<TitleDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [seasonIndex, setSeasonIndex] = useState(0);
  const [favorite, setFavorite] = useState(false);

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
    return () => {
      active = false;
    };
  }, [item]);

  const shown = detail ?? item;
  const seasons = detail?.seasons ?? [];
  const selectedSeason = seasons[seasonIndex];
  const people = detail?.actors ?? detail?.cast ?? [];
  const tags = detail?.genres ?? detail?.categories ?? [];
  const playable = isPlayable(shown);
  const backdrop = useMemo(() => artworkUrl(shown.thumbnail_url), [shown.thumbnail_url]);

  const toggleFavorite = async () => {
    try {
      if (favorite) await userApi.removeFavorite(titleId(shown));
      else await userApi.addFavorite(titleId(shown));
      setFavorite((value) => !value);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to update My List.");
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6" role="dialog" aria-modal="true" aria-label={shown.title ?? "Title details"}>
      <button type="button" aria-label="Close title details" className="absolute inset-0 bg-black/85" onClick={onClose} />
      <article className="relative z-10 max-h-[92vh] w-full max-w-4xl overflow-y-auto rounded-lg bg-[#181818] shadow-2xl">
        <button type="button" onClick={onClose} aria-label="Close" className="absolute right-4 top-4 z-30 flex h-9 w-9 items-center justify-center rounded-full bg-black text-2xl hover:bg-zinc-800">
          ×
        </button>
        <div className="relative h-[42vh] min-h-72 bg-cover bg-center sm:h-[55vh]" style={{ backgroundImage: `url("${backdrop}")` }}>
          <div className="absolute inset-0 bg-linear-to-t from-[#181818] via-transparent to-black/20" />
          <div className="absolute inset-x-5 bottom-8 sm:inset-x-10">
            <h2 className="max-w-2xl text-3xl font-black sm:text-5xl">{shown.title ?? "Title details"}</h2>
            <div className="mt-5 flex gap-3">
              {playable ? <Link href={watchHref(shown.asset_id as string, { kind: shown.progress_kind ?? (shown.movie_id ? "movie" : undefined), id: shown.progress_id ?? shown.movie_id, title: shown.title })} className="flex items-center gap-2 rounded bg-white px-5 py-2 font-bold text-black"><Play size={20} fill="currentColor" /> Play</Link> : <button disabled className="flex items-center gap-2 rounded bg-gray-600 px-5 py-2 font-bold text-gray-300"><Play size={20} fill="currentColor" /> Processing</button>}
              <button type="button" onClick={toggleFavorite} aria-label="Toggle My List" className="flex h-10 w-10 items-center justify-center rounded-full border-2 border-gray-400 hover:border-white">
                <Plus size={24} className={favorite ? "rotate-45" : ""} />
              </button>
            </div>
          </div>
        </div>

        <div className="grid gap-7 px-5 pb-10 sm:px-10 md:grid-cols-[2fr_1fr]">
          <div>
            {loading && <p className="animate-pulse text-gray-300">Loading title information…</p>}
            {error && <p className="rounded bg-red-950/70 p-3 text-sm text-red-100">{error}</p>}
            <div className="mb-4 flex flex-wrap gap-3 text-sm text-gray-300">
              {shown.year_released && <span>{shown.year_released}</span>}
              {detail?.duration && <span>{durationLabel(detail.duration)}</span>}
              <span className="border border-gray-400 px-1">TV-MA</span>
            </div>
            <p className="font-medium leading-relaxed">{shown.description ?? "No description is available yet."}</p>
          </div>
          <dl className="space-y-3 text-sm">
            <div><dt className="inline text-gray-400">Cast: </dt><dd className="inline">{people.join(", ") || "Not available"}</dd></div>
            <div><dt className="inline text-gray-400">Genres: </dt><dd className="inline">{tags.join(", ") || "Not available"}</dd></div>
            {detail?.director && <div><dt className="inline text-gray-400">Director: </dt><dd className="inline">{detail.director}</dd></div>}
          </dl>
        </div>

        {seasons.length > 0 && (
          <section className="px-5 pb-10 sm:px-10">
            <div className="mb-5 flex items-center justify-between gap-4">
              <h3 className="text-2xl font-bold">Episodes</h3>
              <select value={seasonIndex} onChange={(event) => setSeasonIndex(Number(event.target.value))} className="rounded border border-gray-500 bg-black px-3 py-2 text-white">
                {seasons.map((season, index) => (
                  <option key={season.season_id ?? season.id ?? index} value={index}>
                    Season {season.season_number ?? season.number ?? index + 1}
                  </option>
                ))}
              </select>
            </div>
            <div className="divide-y divide-gray-700">
              {(selectedSeason?.episodes ?? []).map((episode, index) => {
                const episodePlayable = Boolean(episode.asset_id) && (!episode.asset_status || episode.asset_status === "ready");
                return (
                  <div key={episode.episode_id ?? episode.id ?? index} className="grid grid-cols-[2rem_1fr] gap-3 py-5 sm:grid-cols-[2rem_10rem_1fr] sm:items-center">
                    <span className="text-xl font-bold">{episode.episode_number ?? episode.episode ?? index + 1}</span>
                    {episode.thumbnail_url && (
                      <div className="relative hidden aspect-video overflow-hidden rounded sm:block">
                        <Image src={artworkUrl(episode.thumbnail_url)} alt="" fill className="object-cover" unoptimized />
                      </div>
                    )}
                    <div>
                      <div className="flex items-center justify-between gap-3"><h4 className="font-bold">{episode.title}</h4><span className="text-sm">{durationLabel(episode.duration)}</span></div>
                      <p className="mt-2 text-sm text-gray-300">{episode.description}</p>
                      {!episodePlayable && <p className="mt-2 text-xs font-semibold text-amber-400">This episode is still processing.</p>}
                      {episodePlayable && <Link href={watchHref(episode.asset_id as string, { kind: "episode", id: episode.episode_id ?? episode.id, title: episode.title })} className="mt-3 inline-flex min-h-10 items-center gap-2 rounded bg-white px-4 py-2 text-sm font-bold text-black"><Play size={17} fill="currentColor" /> Play episode</Link>}
                    </div>
                  </div>
                );
              })}
            </div>
          </section>
        )}
      </article>
    </div>
  );
}
