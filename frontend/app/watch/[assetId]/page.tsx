import { VideoPlayer } from "@/components/VideoPlayer";
import type { ProgressKind } from "@/utils/api/client";

type WatchPageProps = {
  params: Promise<{ assetId: string }>;
  searchParams: Promise<{ kind?: string; id?: string; title?: string }>;
};

export default async function WatchPage({ params, searchParams }: WatchPageProps) {
  const { assetId } = await params;
  const query = await searchParams;
  const kind: ProgressKind | undefined = query.kind === "movie" || query.kind === "episode" ? query.kind : undefined;
  const parsedId = query.id ? Number(query.id) : undefined;
  const progressId = parsedId && Number.isInteger(parsedId) && parsedId > 0 ? parsedId : undefined;

  return <VideoPlayer assetId={assetId} title={query.title} progressKind={kind} progressId={progressId} />;
}
