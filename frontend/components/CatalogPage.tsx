"use client";

import { Carousel } from "@/components/Carousel";
import { Hero } from "@/components/Hero";
import { Navbar } from "@/components/Navbar";
import { TitleModal } from "@/components/TitleModal";
import { catalogApi, userApi, type CatalogItem, type HomeRow } from "@/utils/api/client";
import { useEffect, useMemo, useState } from "react";

export type CatalogMode = "home" | "movies" | "series" | "new" | "favorites";

const labels: Record<CatalogMode, string> = {
  home: "Home",
  movies: "Movies",
  series: "Series",
  new: "Popular new arrivals",
  favorites: "My List",
};

function matchesMode(item: CatalogItem, mode: CatalogMode): boolean {
  const type = item.content_type.toLowerCase();
  if (mode === "movies") return type.includes("movie");
  if (mode === "series") return type.includes("series") || type.includes("show");
  return true;
}

async function loadRows(mode: CatalogMode): Promise<HomeRow[]> {
  if (mode === "favorites") {
    const [favorites, catalog] = await Promise.all([userApi.favorites(), catalogApi.titles("?limit=100&offset=0")]);
    const items = favorites.flatMap((favorite) => {
      const playable = catalog.find((item) => (item.title_id ?? item.id) === favorite.title_id);
      return playable ? [{ ...playable, ...favorite }] : [];
    });
    return [{ id: "favorites", title: "My List", items }];
  }
  if (mode === "new") {
    const items = await catalogApi.titles("?limit=40&offset=0");
    return [{ id: "new-arrivals", title: "Popular new arrivals", items }];
  }

  const rows = await catalogApi.home();
  const filtered = rows
    .map((row) => ({ ...row, items: row.items.filter((item) => matchesMode(item, mode)) }))
    .filter((row) => row.items.length > 0);

  if (mode !== "home") return filtered;

  const progress = await userApi.continueWatching().catch(() => []);
  const catalogItems = filtered.flatMap((row) => row.items);
  const progressItems = progress.flatMap((entry) => {
    const playable = catalogItems.find((item) => (item.title_id ?? item.id) === entry.title_id);
    return playable ? [{ ...playable, title: entry.title, progress_kind: entry.kind, progress_id: entry.content_id, current_time_seconds: entry.current_time_seconds }] : [];
  });
  const withoutContinue = filtered.filter((row) => !row.title.toLowerCase().includes("continue"));
  const catalogContinue = filtered.find((row) => row.title.toLowerCase().includes("continue"));
  if (progressItems.length > 0) return [{ id: "continue", title: "Continue watching", items: progressItems }, ...withoutContinue];
  return catalogContinue ? [catalogContinue, ...withoutContinue] : withoutContinue;
}

export function CatalogPage({ mode }: { mode: CatalogMode }) {
  const [rows, setRows] = useState<HomeRow[]>([]);
  const [selected, setSelected] = useState<CatalogItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    loadRows(mode)
      .then((value) => active && setRows(value))
      .catch((reason: unknown) => {
        if (active) setError(reason instanceof Error ? reason.message : `Unable to load ${labels[mode]}.`);
      })
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [mode]);

  const hero = useMemo(() => rows.flatMap((row) => row.items)[0], [rows]);

  return (
    <main className="min-h-screen bg-black text-white">
      <div className="absolute inset-x-0 top-0 z-40 bg-linear-to-b from-black/90 to-transparent">
        <Navbar />
      </div>
      {loading && (
        <section className="flex min-h-screen items-center justify-center">
          <p className="animate-pulse text-lg">Loading {labels[mode]}…</p>
        </section>
      )}
      {!loading && error && (
        <section className="flex min-h-screen items-center justify-center px-6 text-center">
          <div><h1 className="text-3xl font-bold">We couldn&apos;t load {labels[mode]}</h1><p className="mt-3 text-red-300">{error}</p></div>
        </section>
      )}
      {!loading && !error && !hero && (
        <section className="flex min-h-screen items-center justify-center px-6 text-center">
          <div><h1 className="text-3xl font-bold">Nothing here yet</h1><p className="mt-3 text-gray-400">New ready-to-watch titles will appear here automatically.</p></div>
        </section>
      )}
      {!loading && !error && hero && (
        <>
          <Hero item={hero} onMoreInfo={setSelected} />
          <div className="relative z-10 -mt-20 pb-16">
            {rows.map((row) => <Carousel key={row.id} row={row} onSelect={setSelected} />)}
          </div>
        </>
      )}
      {selected && <TitleModal item={selected} onClose={() => setSelected(null)} />}
    </main>
  );
}
