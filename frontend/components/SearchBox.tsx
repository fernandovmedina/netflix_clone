"use client";

import { TitleModal } from "@/components/TitleModal";
import { artworkUrl, catalogApi, type CatalogItem } from "@/utils/api/client";
import { Search, X } from "@deemlol/next-icons";
import Image from "next/image";
import { useEffect, useRef, useState } from "react";

const DEBOUNCE_MS = 300;

function kindLabel(item: CatalogItem): string {
  return item.content_type.toLowerCase().includes("movie") ? "Movie" : "Series";
}

export function SearchBox() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<CatalogItem[]>([]);
  const [searching, setSearching] = useState(false);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<CatalogItem | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) inputRef.current?.focus();
  }, [open]);

  // Search by name only; the catalog endpoint matches `q` against the title.
  useEffect(() => {
    const name = query.trim();
    if (!open || name.length === 0) {
      setResults([]);
      setSearching(false);
      setError("");
      return;
    }
    let active = true;
    setSearching(true);
    const timer = window.setTimeout(() => {
      catalogApi.search(name)
        .then((items) => { if (active) { setResults(items); setError(""); } })
        .catch((reason: unknown) => { if (active) setError(reason instanceof Error ? reason.message : "Search is unavailable right now."); })
        .finally(() => { if (active) setSearching(false); });
    }, DEBOUNCE_MS);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [open, query]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      if (query) setQuery("");
      else setOpen(false);
    };
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open, query]);

  const choose = (item: CatalogItem) => {
    setSelected(item);
    setOpen(false);
    setQuery("");
  };

  const name = query.trim();

  return <>
    <div ref={containerRef} className="relative min-w-0">
      {open ? (
        <form role="search" onSubmit={(event) => event.preventDefault()} className="flex min-w-0 items-center gap-1 rounded border border-white/40 bg-black/80 px-2 sm:gap-2">
          <Search size={20} />
          <input
            ref={inputRef}
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Titles"
            aria-label="Search movies and series by name"
            className="min-h-11 w-28 min-w-0 bg-transparent text-base outline-none placeholder:text-gray-400 sm:w-56"
          />
          <button type="button" onClick={() => { setQuery(""); setOpen(false); }} aria-label="Close search" className="flex h-11 w-8 items-center justify-center text-gray-300 hover:text-white"><X size={18} /></button>
        </form>
      ) : (
        <button type="button" onClick={() => setOpen(true)} aria-label="Search movies and series by name" aria-expanded={false} className="flex h-11 w-11 items-center justify-center rounded hover:bg-white/10">
          <Search size={24} />
        </button>
      )}

      {open && name.length > 0 && <div className="absolute right-0 top-full z-50 mt-2 max-h-[70vh] w-[min(24rem,calc(100vw-2rem))] overflow-y-auto rounded-lg border border-zinc-700 bg-black/95 p-2 shadow-2xl">
        {searching && results.length === 0 && <p className="px-3 py-4 text-sm text-gray-400">Searching…</p>}
        {error && <p className="px-3 py-4 text-sm text-red-300">{error}</p>}
        {!searching && !error && results.length === 0 && <p className="px-3 py-4 text-sm text-gray-400">No titles match “{name}”.</p>}
        {results.map((item) => <button
          type="button"
          key={item.title_id ?? item.id}
          onClick={() => choose(item)}
          className="flex w-full items-center gap-3 rounded px-2 py-2 text-left hover:bg-zinc-800"
        >
          <span className="relative aspect-video w-20 shrink-0 overflow-hidden rounded bg-zinc-800">
            {item.thumbnail_url && <Image src={artworkUrl(item.thumbnail_url)} alt="" fill sizes="5rem" className="object-cover" unoptimized />}
          </span>
          <span className="min-w-0 flex-1">
            <span className="block truncate text-sm font-semibold">{item.title ?? `Title ${item.id}`}</span>
            <span className="block text-xs text-gray-400">{kindLabel(item)}{item.year_released ? ` · ${item.year_released}` : ""}</span>
          </span>
        </button>)}
      </div>}
    </div>

    {selected && <TitleModal item={selected} onClose={() => setSelected(null)} />}
  </>;
}
