import { VideoPlayer } from "@/components/VideoPlayer";
import type { ProgressKind } from "@/utils/api/client";
import { pageMetadata } from "@/utils/metadata";
import type { Metadata } from "next";

type WatchPageProps = {
  params: Promise<{ assetId: string }>;
  searchParams: Promise<{ kind?: string; id?: string; title?: string }>;
};

export async function generateMetadata({ searchParams }: WatchPageProps): Promise<Metadata> {
  const title = (await searchParams).title?.trim();
  return pageMetadata(title ? `Watching ${title}` : "Watch", title ? `Now playing ${title}.` : "Watch your selected title.", { authenticated: true });
}

export default async function WatchPage({ params, searchParams }: WatchPageProps) {
  const { assetId } = await params;
  const query = await searchParams;
  const kind: ProgressKind | undefined = query.kind === "movie" || query.kind === "episode" ? query.kind : undefined;
  const parsedId = query.id ? Number(query.id) : undefined;
  const progressId = parsedId && Number.isInteger(parsedId) && parsedId > 0 ? parsedId : undefined;

  return <VideoPlayer assetId={assetId} title={query.title} progressKind={kind} progressId={progressId} />;
}
