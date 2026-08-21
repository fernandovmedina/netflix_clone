"use client";

import { UploadPanel } from "@/components/admin/UploadPanel";
import { adminApi, apiRequest, type AdminTitleInput, type Season, type TitleDetail } from "@/utils/api/client";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

type EditorKind = "movies" | "series";
type AdminTitleEditorProps = { kind: EditorKind; titleId?: number };

const emptyForm: AdminTitleInput = { title: "", description: "", director: "", year_released: new Date().getFullYear(), duration: 0, number_of_seasons: 1 };

export function AdminTitleEditor({ kind, titleId }: AdminTitleEditorProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [form, setForm] = useState<AdminTitleInput>(emptyForm);
  const [entityId, setEntityId] = useState<number | null>(() => {
    const queryId = Number(searchParams.get("entityId"));
    return Number.isInteger(queryId) && queryId > 0 ? queryId : titleId ?? null;
  });
  const [resolvedTitleId, setResolvedTitleId] = useState<number | null>(titleId ?? null);
  const [detail, setDetail] = useState<TitleDetail | null>(null);
  const [loading, setLoading] = useState(Boolean(titleId));
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState("");
  const [error, setError] = useState("");
  const [assetReady, setAssetReady] = useState(false);

  const load = useCallback(async () => {
    if (!titleId) return;
    setLoading(true);
    setError("");
    try {
      const value = await apiRequest<TitleDetail>(`/api/v1/${kind}/${titleId}`);
      setDetail(value);
      setResolvedTitleId(value.title_id ?? value.id);
      if (kind === "series" && value.series_id) setEntityId(value.series_id);
      if (kind === "movies" && value.movie_id) setEntityId(value.movie_id);
      setForm({
        title: value.title ?? "",
        description: value.description ?? "",
        director: value.director ?? "",
        year_released: value.year_released ?? new Date().getFullYear(),
        duration: typeof value.duration === "number" ? value.duration : 0,
        number_of_seasons: value.seasons?.length ?? 1,
      });
      setAssetReady(Boolean(value.asset_id));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to load this title.");
    } finally {
      setLoading(false);
    }
  }, [kind, titleId]);

  useEffect(() => { load(); }, [load]);

  const setField = (field: keyof AdminTitleInput, value: string | number) => setForm((current) => ({ ...current, [field]: value }));

  const save = async (event: React.FormEvent) => {
    event.preventDefault();
    setSaving(true);
    setSaved("");
    setError("");
    try {
      if (!resolvedTitleId) {
        const created = await adminApi.createTitle(kind, form);
        setEntityId(created.id);
        setResolvedTitleId(created.title_id);
        setSaved("Metadata created. Add artwork and video before publishing.");
        router.replace(`/admin/${kind}/${created.title_id}?entityId=${created.id}`);
      } else if (entityId) {
        await adminApi.updateTitle(kind, entityId, form);
        setSaved("Changes saved.");
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to save this title.");
    } finally {
      setSaving(false);
    }
  };

  const togglePublished = async () => {
    if (!resolvedTitleId || (!detail?.published && !assetReady)) return;
    try {
      const next = !detail?.published;
      await adminApi.publish(resolvedTitleId, next);
      setDetail((current) => current ? { ...current, published: next } : ({ ...emptyForm, id: resolvedTitleId, content_type: kind === "movies" ? "Movie" : "TV Show", thumbnail_url: "", published: next } as TitleDetail));
      setSaved(next ? "Title published." : "Title unpublished.");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to change publication status.");
    }
  };

  if (loading) return <p className="animate-pulse text-zinc-300">Loading title…</p>;

  return (
    <div className="mx-auto max-w-5xl">
      <div className="mb-7 flex flex-wrap items-center justify-between gap-4">
        <div><Link href="/admin" className="text-sm text-zinc-400 hover:text-white">← All titles</Link><h1 className="mt-2 text-3xl font-black sm:text-4xl">{resolvedTitleId ? "Edit" : "Create"} {kind === "movies" ? "movie" : "series"}</h1></div>
        {resolvedTitleId && <button type="button" onClick={togglePublished} disabled={!detail?.published && !assetReady} className="min-h-11 rounded border border-zinc-500 px-5 font-bold hover:border-white disabled:cursor-not-allowed disabled:opacity-40">{detail?.published ? "Unpublish" : "Publish"}</button>}
      </div>
      {resolvedTitleId && !assetReady && !detail?.published && <p className="mb-5 rounded border border-amber-700/50 bg-amber-950/30 p-4 text-sm text-amber-100">Publishing is locked until video processing reaches Ready.</p>}
      {saved && <p className="mb-5 rounded bg-emerald-950/60 p-4 text-emerald-200">{saved}</p>}
      {error && <p className="mb-5 rounded bg-red-950/60 p-4 text-red-200">{error}</p>}

      <form onSubmit={save} className="grid gap-5 rounded-xl border border-zinc-700 bg-zinc-900/60 p-5 sm:grid-cols-2 sm:p-7">
        <label className="sm:col-span-2">Title<input required value={form.title} onChange={(event) => setField("title", event.target.value)} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>
        <label>Release year<input type="number" min={1888} max={2200} value={form.year_released} onChange={(event) => setField("year_released", Number(event.target.value))} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>
        <label>Director<input value={form.director} onChange={(event) => setField("director", event.target.value)} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>
        {kind === "movies" ? <label>Duration (minutes)<input type="number" min={0} value={form.duration} onChange={(event) => setField("duration", Number(event.target.value))} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label> : <label>Initial number of seasons<input type="number" min={0} max={100} value={form.number_of_seasons} onChange={(event) => setField("number_of_seasons", Number(event.target.value))} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>}
        <label className="sm:col-span-2">Description<textarea rows={5} value={form.description} onChange={(event) => setField("description", event.target.value)} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>
        <button disabled={saving} className="min-h-12 rounded bg-red-600 px-6 font-bold hover:bg-red-700 disabled:opacity-50 sm:col-span-2">{saving ? "Saving…" : resolvedTitleId ? "Save changes" : "Create metadata"}</button>
      </form>

      {resolvedTitleId && <div className="mt-7 grid gap-5">
        <UploadPanel endpoint={`/api/v1/admin/titles/${resolvedTitleId}/thumbnail`} label="Poster artwork" accept="image/jpeg,image/png,image/webp" pollAsset={false} onUploaded={load} />
        {kind === "movies" && entityId && <UploadPanel endpoint={`/api/v1/admin/movies/${entityId}/video`} label="Movie video" accept="video/mp4,video/quicktime,video/webm,video/x-matroska,.mkv" initialReadyAssetId={detail?.asset_id} onReady={() => setAssetReady(true)} />}
      </div>}

      {kind === "series" && entityId && resolvedTitleId && <SeriesManager seriesId={entityId} seasons={detail?.seasons ?? []} onReload={load} onAssetReady={() => setAssetReady(true)} />}
    </div>
  );
}

function SeriesManager({ seriesId, seasons, onReload, onAssetReady }: { seriesId: number; seasons: Season[]; onReload: () => Promise<void>; onAssetReady: () => void }) {
  const [seasonNumber, setSeasonNumber] = useState(seasons.length + 1);
  const [episodeTitles, setEpisodeTitles] = useState<Record<number, string>>({});
  const [error, setError] = useState("");

  const addSeason = async () => {
    try { await adminApi.createSeason(seriesId, seasonNumber); await onReload(); setSeasonNumber((value) => value + 1); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Unable to create season."); }
  };

  const addEpisode = async (season: Season) => {
    const seasonId = season.season_id ?? season.id;
    const title = seasonId ? episodeTitles[seasonId]?.trim() : "";
    if (!seasonId || !title) return;
    try { await adminApi.createEpisode(seasonId, season.episodes.length + 1, title); setEpisodeTitles((current) => ({ ...current, [seasonId]: "" })); await onReload(); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Unable to create episode."); }
  };

  return <section className="mt-8 rounded-xl border border-zinc-700 bg-zinc-900/60 p-5 sm:p-7">
    <div className="flex flex-wrap items-end justify-between gap-4"><div><h2 className="text-2xl font-black">Seasons and episodes</h2><p className="mt-1 text-sm text-zinc-400">Create episodes, then upload and monitor each episode&apos;s video independently.</p></div><div className="flex gap-2"><input aria-label="Season number" type="number" min={1} value={seasonNumber} onChange={(event) => setSeasonNumber(Number(event.target.value))} className="w-24 rounded border border-zinc-600 bg-black px-3" /><button type="button" onClick={addSeason} className="min-h-11 rounded bg-white px-4 font-bold text-black">Add season</button></div></div>
    {error && <p className="mt-4 text-red-300">{error}</p>}
    <div className="mt-6 grid gap-6">
      {seasons.map((season, index) => {
        const seasonId = season.season_id ?? season.id;
        return <article key={seasonId ?? index} className="rounded-lg border border-zinc-700 bg-black/50 p-4"><h3 className="text-xl font-bold">Season {season.season_number ?? season.number ?? index + 1}</h3>
          <div className="mt-4 grid gap-4">{season.episodes.map((episode, episodeIndex) => <div key={episode.episode_id ?? episode.id ?? episodeIndex} className="rounded border border-zinc-800 p-4"><p className="font-bold">{episode.episode_number ?? episode.episode ?? episodeIndex + 1}. {episode.title}</p>{(episode.episode_id ?? episode.id) && <div className="mt-3"><UploadPanel endpoint={`/api/v1/admin/episodes/${episode.episode_id ?? episode.id}/video`} label="Episode video" accept="video/mp4,video/quicktime,video/webm,video/x-matroska,.mkv" initialReadyAssetId={episode.asset_id} onReady={onAssetReady} /></div>}</div>)}</div>
          {seasonId && <div className="mt-4 flex flex-col gap-2 sm:flex-row"><input value={episodeTitles[seasonId] ?? ""} onChange={(event) => setEpisodeTitles((current) => ({ ...current, [seasonId]: event.target.value }))} placeholder="New episode title" className="min-h-11 flex-1 rounded border border-zinc-600 bg-black px-4" /><button type="button" onClick={() => addEpisode(season)} className="min-h-11 rounded border border-white px-4 font-bold">Add episode</button></div>}
        </article>;
      })}
    </div>
  </section>;
}
