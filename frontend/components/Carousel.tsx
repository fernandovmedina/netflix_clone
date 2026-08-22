"use client";

import { artworkUrl, type HomeRow } from "@/utils/api/client";
import Image from "next/image";
import { useRef } from "react";

type CarouselProps = {
  row: HomeRow;
  onSelect: (item: HomeRow["items"][number]) => void;
};

export function Carousel({ row, onSelect }: CarouselProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const scroll = (direction: -1 | 1) => {
    const width = scrollRef.current?.clientWidth ?? 900;
    scrollRef.current?.scrollBy({ left: direction * width * 0.85, behavior: "smooth" });
  };

  if (row.items.length === 0) return null;

  return (
    <section className="my-8" aria-labelledby={`row-${row.id}`}>
      <h2 id={`row-${row.id}`} className="mb-3 px-5 text-lg font-bold sm:px-10 sm:text-xl lg:px-14">
        {row.title}
      </h2>
      <div className="group relative">
        <button
          type="button"
          aria-label={`Scroll ${row.title} left`}
          onClick={() => scroll(-1)}
          className="absolute left-0 top-1/2 z-20 hidden h-16 w-10 -translate-y-1/2 items-center justify-center bg-black/70 text-3xl hover:bg-black/90 sm:group-hover:flex sm:w-12"
        >
          ‹
        </button>
        <div ref={scrollRef} className="flex snap-x snap-mandatory touch-pan-x gap-2 overflow-x-auto overscroll-x-contain scroll-px-5 px-5 pb-3 sm:scroll-px-10 sm:px-10 lg:scroll-px-14 lg:px-14">
          {row.items.map((item, index) => (
            <button
              type="button"
              key={`${item.title_id ?? item.id}-${index}`}
              onClick={() => onSelect(item)}
              className="relative aspect-video w-[72vw] max-w-[310px] shrink-0 snap-start overflow-hidden rounded-md bg-zinc-800 text-left transition duration-300 hover:z-10 hover:scale-105 sm:w-[38vw] md:w-[28vw] lg:w-[22vw] xl:w-[18vw]"
            >
              <Image
                src={artworkUrl(item.thumbnail_url)}
                alt={item.title ?? `Title ${item.title_id ?? item.id}`}
                fill
                sizes="(max-width: 640px) 72vw, (max-width: 1024px) 28vw, 18vw"
                className="object-cover"
                unoptimized
              />
              {item.title && (
                <span className="absolute inset-x-0 bottom-0 bg-linear-to-t from-black/90 to-transparent px-3 pb-2 pt-8 text-sm font-semibold">
                  {item.title}
                </span>
              )}
            </button>
          ))}
        </div>
        <button
          type="button"
          aria-label={`Scroll ${row.title} right`}
          onClick={() => scroll(1)}
          className="absolute right-0 top-1/2 z-20 hidden h-16 w-10 -translate-y-1/2 items-center justify-center bg-black/70 text-3xl hover:bg-black/90 sm:group-hover:flex sm:w-12"
        >
          ›
        </button>
      </div>
    </section>
  );
}
