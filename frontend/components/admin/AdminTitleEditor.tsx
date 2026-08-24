"use client";

import { UploadPanel } from "@/components/admin/UploadPanel";
import { adminApi, apiRequest, catalogApi, type AdminTitleInput, type MetadataReference, type Season, type TitleDetail } from "@/utils/api/client";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

type EditorKind = "movies" | "series";
type AdminTitleEditorProps = { kind: EditorKind; titleId?: number; initialDetail?: TitleDetail };

const emptyForm: AdminTitleInput = { title: "", description: "", director: "", year_released: new Date().getFullYear(), duration: 0, number_of_seasons: 1, genre_ids: [], actor_ids: [] };

function formFromDetail(value: TitleDetail): AdminTitleInput {
  return {
    title: value.title ?? "",
    description: value.description ?? "",
    director: value.director ?? "",
    year_released: value.year_released ?? new Date().getFullYear(),
    duration: typeof value.duration === "number" ? value.duration : 0,
    number_of_seasons: value.seasons?.length ?? 1,
    genre_ids: [],
    actor_ids: [],
  };
}

function detailHasReadyAsset(kind: EditorKind, value?: TitleDetail): boolean {
  if (!value) return false;
  return kind === "movies"
    ? value.asset_status === "ready"
    : value.seasons?.some((season) => season.episodes.some((episode) => episode.asset_status === "ready")) ?? false;
}

export function AdminTitleEditor({ kind, titleId, initialDetail }: AdminTitleEditorProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [form, setForm] = useState<AdminTitleInput>(() => initialDetail ? formFromDetail(initialDetail) : emptyForm);
  const [entityId, setEntityId] = useState<number | null>(() => {
    const queryId = Number(searchParams.get("entityId"));
    if (Number.isInteger(queryId) && queryId > 0) return queryId;
    if (kind === "series" && initialDetail?.series_id) return initialDetail.series_id;
    if (kind === "movies" && initialDetail?.movie_id) return initialDetail.movie_id;
    return titleId ?? null;
  });
  const [resolvedTitleId, setResolvedTitleId] = useState<number | null>(initialDetail?.title_id ?? titleId ?? null);
  const [detail, setDetail] = useState<TitleDetail | null>(initialDetail ?? null);
  const [loading, setLoading] = useState(Boolean(titleId && !initialDetail));
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState("");
  const [error, setError] = useState("");
  const [assetReady, setAssetReady] = useState(() => detailHasReadyAsset(kind, initialDetail));
  const [genres, setGenres] = useState<MetadataReference[]>([]);
  const [actors, setActors] = useState<MetadataReference[]>([]);
  const [metadataLoading, setMetadataLoading] = useState(true);
  const [metadataError, setMetadataError] = useState("");
  const [associationDirty, setAssociationDirty] = useState({ genre_ids: false, actor_ids: false });

  useEffect(() => {
    let active = true;
    setMetadataLoading(true);
    Promise.all([catalogApi.genres(), catalogApi.actors()])
      .then(([genreOptions, actorOptions]) => {
        if (!active) return;
        setGenres(genreOptions);
        setActors(actorOptions);
        setMetadataError("");
      })
      .catch((reason: unknown) => {
        if (active) setMetadataError(reason instanceof Error ? reason.message : "Unable to load genres and cast.");
      })
      .finally(() => {
        if (active) setMetadataLoading(false);
      });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!detail || metadataLoading) return;
    setForm((current) => ({
      ...current,
      genre_ids: genres.filter((option) => detail.genres?.includes(option.name)).map((option) => option.id),
      actor_ids: actors.filter((option) => (detail.actors ?? detail.cast)?.includes(option.name)).map((option) => option.id),
    }));
    setAssociationDirty({ genre_ids: false, actor_ids: false });
  }, [actors, detail, genres, metadataLoading]);

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
      setForm(formFromDetail(value));
      setAssetReady(detailHasReadyAsset(kind, value));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to load this title.");
    } finally {
      setLoading(false);
    }
  }, [kind, titleId]);

  useEffect(() => { if (!initialDetail) load(); }, [initialDetail, load]);

  const setField = (field: keyof AdminTitleInput, value: string | number) => setForm((current) => ({ ...current, [field]: value }));
  const toggleMetadata = (field: "genre_ids" | "actor_ids", id: number) => setForm((current) => {
    const selected = current[field] ?? [];
    return { ...current, [field]: selected.includes(id) ? selected.filter((value) => value !== id) : [...selected, id] };
  });

  const changeMetadata = (field: "genre_ids" | "actor_ids", id: number) => {
    setAssociationDirty((current) => ({ ...current, [field]: true }));
    toggleMetadata(field, id);
  };

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
        const routeId = kind === "movies" ? created.id : created.title_id;
        router.replace(`/admin/${kind}/${routeId}?entityId=${created.id}`);
      } else if (entityId) {
        const payload: AdminTitleInput = { ...form };
        if (!associationDirty.genre_ids) delete payload.genre_ids;
        if (!associationDirty.actor_ids) delete payload.actor_ids;
        delete payload.category_ids;
        await adminApi.updateTitle(kind, entityId, payload);
        await load();
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
        <div className="min-w-0"><Link href="/admin" className="text-sm text-zinc-400 hover:text-white">← All titles</Link><h1 className="mt-2 text-3xl font-black sm:text-4xl">{resolvedTitleId ? "Edit" : "Create"} {kind === "movies" ? "movie" : "series"}</h1></div>
        {resolvedTitleId && <button type="button" onClick={togglePublished} disabled={!detail?.published && !assetReady} className="min-h-11 w-full whitespace-nowrap rounded border border-zinc-500 px-5 font-bold hover:border-white disabled:cursor-not-allowed disabled:opacity-40 sm:w-auto">{detail?.published ? "Unpublish" : "Publish"}</button>}
      </div>
      {resolvedTitleId && !assetReady && !detail?.published && <p className="mb-5 rounded border border-amber-700/50 bg-amber-950/30 p-4 text-sm text-amber-100">Publishing is locked until video processing reaches Ready.</p>}
      {saved && <p className="mb-5 rounded bg-emerald-950/60 p-4 text-emerald-200">{saved}</p>}
      {error && <p className="mb-5 rounded bg-red-950/60 p-4 text-red-200">{error}</p>}

      <form onSubmit={save} className="grid min-w-0 gap-5 rounded-xl border border-zinc-700 bg-zinc-900/60 p-5 sm:grid-cols-2 sm:p-7">
        <label className="sm:col-span-2">Title<input required value={form.title} onChange={(event) => setField("title", event.target.value)} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>
        <label>Release year<input type="number" min={1888} max={2200} value={form.year_released} onChange={(event) => setField("year_released", Number(event.target.value))} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>
        <label>Director<input value={form.director} onChange={(event) => setField("director", event.target.value)} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>
        {kind === "movies" ? <label>Duration (minutes)<input type="number" min={0} value={form.duration} onChange={(event) => setField("duration", Number(event.target.value))} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label> : <label>Initial number of seasons<input type="number" min={0} max={100} value={form.number_of_seasons} onChange={(event) => setField("number_of_seasons", Number(event.target.value))} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>}
        <label className="sm:col-span-2">Description<textarea rows={5} value={form.description} onChange={(event) => setField("description", event.target.value)} className="mt-2 w-full rounded border border-zinc-600 bg-black px-4 py-3" /></label>
        <MetadataChoices legend="Genres" options={genres} selected={form.genre_ids ?? []} loading={metadataLoading} onToggle={(id) => changeMetadata("genre_ids", id)} />
        <MetadataChoices legend="Cast" options={actors} selected={form.actor_ids ?? []} loading={metadataLoading} onToggle={(id) => changeMetadata("actor_ids", id)} />
        {metadataError && <p role="alert" className="rounded bg-red-950/60 p-4 text-red-200 sm:col-span-2">{metadataError}</p>}
        <button disabled={saving || metadataLoading || Boolean(metadataError)} className="min-h-12 rounded bg-red-600 px-6 font-bold hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50 sm:col-span-2">{saving ? "Saving…" : metadataLoading ? "Loading metadata…" : resolvedTitleId ? "Save changes" : "Create metadata"}</button>
      </form>

      {resolvedTitleId && <div className="mt-7 grid gap-5">
        <UploadPanel endpoint={`/api/v1/admin/titles/${resolvedTitleId}/thumbnail`} label="Poster artwork" accept="image/jpeg,image/png,image/webp" pollAsset={false} onUploaded={load} />
        {kind === "movies" && entityId && <UploadPanel endpoint={`/api/v1/admin/movies/${entityId}/video`} label="Movie video" accept="video/mp4,video/quicktime,video/webm,video/x-matroska,.mkv" initialAssetId={detail?.asset_id} onReady={() => setAssetReady(true)} />}
      </div>}

      {kind === "series" && entityId && resolvedTitleId && <SeriesManager seriesId={entityId} seasons={detail?.seasons ?? []} onReload={load} onAssetReady={() => setAssetReady(true)} />}
    </div>
  );
}

function MetadataChoices({ legend, options, selected, loading, onToggle }: { legend: string; options: MetadataReference[]; selected: number[]; loading: boolean; onToggle: (id: number) => void }) {
  return <fieldset className="min-w-0 rounded border border-zinc-700 p-4">
    <legend className="px-1 font-bold">{legend} <span className="font-normal text-zinc-400">({selected.length} selected)</span></legend>
    {loading ? <p className="text-sm text-zinc-400">Loading {legend.toLowerCase()}…</p> : options.length === 0 ? <p className="text-sm text-zinc-400">No options are available.</p> : <div className="grid max-h-52 gap-1 overflow-y-auto pr-1">
      {options.map((option) => <label key={option.id} className="flex min-h-10 cursor-pointer items-center gap-3 rounded px-2 hover:bg-zinc-800"><input type="checkbox" checked={selected.includes(option.id)} onChange={() => onToggle(option.id)} className="h-4 w-4 accent-red-600" /><span className="min-w-0 break-words text-sm">{option.name}</span></label>)}
    </div>}
  </fieldset>;
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
    <div className="flex flex-wrap items-end justify-between gap-4"><div className="min-w-0"><h2 className="text-2xl font-black">Seasons and episodes</h2><p className="mt-1 text-sm text-zinc-400">Create episodes, then upload and monitor each episode&apos;s video independently.</p></div><div className="grid w-full grid-cols-[minmax(0,1fr)_auto] gap-2 sm:flex sm:w-auto"><input aria-label="Season number" type="number" min={1} value={seasonNumber} onChange={(event) => setSeasonNumber(Number(event.target.value))} className="min-h-11 min-w-0 rounded border border-zinc-600 bg-black px-3 sm:w-24" /><button type="button" onClick={addSeason} className="min-h-11 whitespace-nowrap rounded bg-white px-4 font-bold text-black">Add season</button></div></div>
    {error && <p className="mt-4 text-red-300">{error}</p>}
    <div className="mt-6 grid gap-6">
      {seasons.map((season, index) => {
        const seasonId = season.season_id ?? season.id;
        return <article key={seasonId ?? index} className="min-w-0 rounded-lg border border-zinc-700 bg-black/50 p-4"><h3 className="text-xl font-bold">Season {season.season_number ?? season.number ?? index + 1}</h3>
          <div className="mt-4 grid min-w-0 gap-4">{season.episodes.map((episode, episodeIndex) => <div key={episode.episode_id ?? episode.id ?? episodeIndex} className="min-w-0 rounded border border-zinc-800 p-4"><p className="break-words font-bold">{episode.episode_number ?? episode.episode ?? episodeIndex + 1}. {episode.title}</p>{(episode.episode_id ?? episode.id) && <div className="mt-3 min-w-0"><UploadPanel endpoint={`/api/v1/admin/episodes/${episode.episode_id ?? episode.id}/video`} label="Episode video" accept="video/mp4,video/quicktime,video/webm,video/x-matroska,.mkv" initialAssetId={episode.asset_id} onReady={onAssetReady} /></div>}</div>)}</div>
          {seasonId && <div className="mt-4 flex flex-col gap-2 sm:flex-row"><input value={episodeTitles[seasonId] ?? ""} onChange={(event) => setEpisodeTitles((current) => ({ ...current, [seasonId]: event.target.value }))} placeholder="New episode title" className="min-h-11 flex-1 rounded border border-zinc-600 bg-black px-4" /><button type="button" onClick={() => addEpisode(season)} className="min-h-11 rounded border border-white px-4 font-bold">Add episode</button></div>}
        </article>;
      })}
    </div>
  </section>;
}
