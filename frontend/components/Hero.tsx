"use client";

import { artworkUrl, isPlayable, watchHref, type CatalogItem } from "@/utils/api/client";
import { Info, Play } from "@deemlol/next-icons";
import Link from "next/link";

type HeroProps = {
  item: CatalogItem;
  onMoreInfo: (item: CatalogItem) => void;
};

export function Hero({ item, onMoreInfo }: HeroProps) {
  const playable = isPlayable(item);

  return (
    <section
      className="relative flex min-h-[72svh] items-end bg-cover bg-center px-5 pb-24 pt-28 sm:min-h-[78svh] sm:px-10 sm:pb-28 lg:min-h-[88svh] lg:px-20 lg:pb-36"
      style={{ backgroundImage: `url("${artworkUrl(item.thumbnail_url)}")` }}
    >
      <div className="absolute inset-0 bg-linear-to-r from-black via-black/55 to-transparent" />
      <div className="absolute inset-0 bg-linear-to-t from-black via-transparent to-black/20" />
      <div className="relative z-10 w-full max-w-xl">
        <p className="mb-3 text-sm font-bold uppercase tracking-[0.3em] text-red-500">
          Featured now
        </p>
        <h1 className="text-3xl font-black drop-shadow-lg sm:text-5xl lg:text-6xl">
          {item.title ?? "Ready to watch"}
        </h1>
        {item.description && (
          <p className="mt-5 line-clamp-3 text-sm font-medium text-gray-100 sm:text-base">
            {item.description}
          </p>
        )}
        <div className="mt-6 flex flex-wrap gap-3">
          {playable ? <Link
            href={watchHref(item.asset_id as string, {
              kind: item.progress_kind ?? (item.movie_id ? "movie" : undefined),
              id: item.progress_id ?? item.movie_id,
              title: item.title,
            })}
            title={playable ? "Play title" : "This title is still processing"}
            className="flex items-center gap-2 rounded bg-white px-6 py-2.5 font-bold text-black hover:bg-white/75 disabled:cursor-not-allowed disabled:bg-gray-600 disabled:text-gray-300"
          >
            <Play size={22} fill="currentColor" />
            Play
          </Link> : <button type="button" disabled title="This title is still processing" className="flex items-center gap-2 rounded bg-gray-600 px-6 py-2.5 font-bold text-gray-300"><Play size={22} fill="currentColor" />Processing</button>}
          <button
            type="button"
            onClick={() => onMoreInfo(item)}
            className="flex min-h-11 items-center gap-2 rounded bg-gray-500/70 px-4 py-2.5 font-bold text-white hover:bg-gray-500/50 sm:px-6"
          >
            <Info size={22} /> More information
          </button>
        </div>
      </div>
    </section>
  );
}
