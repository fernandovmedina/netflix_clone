"use client";

import { artworkUrl, isPlayable, type CatalogItem } from "@/utils/api/client";
import { Info, Play } from "@deemlol/next-icons";

type HeroProps = {
  item: CatalogItem;
  onMoreInfo: (item: CatalogItem) => void;
};

export function Hero({ item, onMoreInfo }: HeroProps) {
  const playable = isPlayable(item);

  return (
    <section
      className="relative min-h-[72vh] bg-cover bg-center px-5 pb-16 pt-64 sm:px-10 lg:min-h-[88vh] lg:px-20 lg:pt-80"
      style={{ backgroundImage: `url("${artworkUrl(item.thumbnail_url)}")` }}
    >
      <div className="absolute inset-0 bg-linear-to-r from-black via-black/55 to-transparent" />
      <div className="absolute inset-0 bg-linear-to-t from-black via-transparent to-black/20" />
      <div className="relative z-10 max-w-xl">
        <p className="mb-3 text-sm font-bold uppercase tracking-[0.3em] text-red-500">
          Featured now
        </p>
        <h1 className="text-4xl font-black drop-shadow-lg sm:text-6xl">
          {item.title ?? "Ready to watch"}
        </h1>
        {item.description && (
          <p className="mt-5 line-clamp-3 text-sm font-medium text-gray-100 sm:text-base">
            {item.description}
          </p>
        )}
        <div className="mt-6 flex flex-wrap gap-3">
          <button
            type="button"
            disabled={!playable}
            title={playable ? "Play title" : "This title is still processing"}
            className="flex items-center gap-2 rounded bg-white px-6 py-2.5 font-bold text-black hover:bg-white/75 disabled:cursor-not-allowed disabled:bg-gray-600 disabled:text-gray-300"
          >
            <Play size={22} fill="currentColor" />
            {playable ? "Play" : "Processing"}
          </button>
          <button
            type="button"
            onClick={() => onMoreInfo(item)}
            className="flex items-center gap-2 rounded bg-gray-500/70 px-6 py-2.5 font-bold text-white hover:bg-gray-500/50"
          >
            <Info size={22} /> More information
          </button>
        </div>
      </div>
    </section>
  );
}
