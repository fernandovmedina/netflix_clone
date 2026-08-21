"use client";

import { API_URL, ApiError, authApi, userApi, type ProgressKind } from "@/utils/api/client";
import Hls from "hls.js";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";

type Quality = { index: number; label: string };

type VideoPlayerProps = {
  assetId: string;
  title?: string;
  progressKind?: ProgressKind;
  progressId?: number;
};

function timeLabel(value: number): string {
  if (!Number.isFinite(value) || value < 0) return "0:00";
  const seconds = Math.floor(value % 60).toString().padStart(2, "0");
  const minutes = Math.floor(value / 60);
  return `${minutes}:${seconds}`;
}

export function VideoPlayer({ assetId, title, progressKind, progressId }: VideoPlayerProps) {
  const router = useRouter();
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const recoveryAttempts = useRef({ network: 0, media: 0 });
  const [playing, setPlaying] = useState(false);
  const [muted, setMuted] = useState(false);
  const [volume, setVolume] = useState(1);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [qualities, setQualities] = useState<Quality[]>([]);
  const [quality, setQuality] = useState(-1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const persistProgress = useCallback(() => {
    const video = videoRef.current;
    if (!video || !progressKind || !progressId || video.currentTime < 1) return;
    userApi.updateProgress(progressKind, progressId, video.currentTime).catch((reason: unknown) => {
      if (reason instanceof ApiError && reason.status === 404) return;
    });
  }, [progressId, progressKind]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    let active = true;
    let resumeAt = 0;
    const source = `${API_URL}/api/v1/stream/${assetId}/master.m3u8`;

    const loadSource = async () => {
      if (progressKind && progressId) {
        try {
          const progress = await userApi.progress(progressKind, progressId);
          resumeAt = progress.current_time_seconds ?? 0;
        } catch (reason) {
          if (!(reason instanceof ApiError) || reason.status !== 404) {
            // Progress is optional; playback remains available when user APIs lag.
          }
        }
      }
      if (!active) return;

      video.crossOrigin = "use-credentials";
      if (video.canPlayType("application/vnd.apple.mpegurl")) {
        video.src = source;
        video.addEventListener("loadedmetadata", () => {
          if (resumeAt > 0 && resumeAt < video.duration * 0.95) video.currentTime = resumeAt;
          setLoading(false);
        }, { once: true });
        return;
      }

      if (!Hls.isSupported()) {
        setError("This browser cannot play HLS video.");
        setLoading(false);
        return;
      }

      const hls = new Hls({
        startLevel: -1,
        capLevelToPlayerSize: true,
        xhrSetup: (xhr) => {
          xhr.withCredentials = true;
        },
      });
      hlsRef.current = hls;
      hls.attachMedia(video);
      hls.on(Hls.Events.MEDIA_ATTACHED, () => hls.loadSource(source));
      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        setQualities(hls.levels.map((level, index) => ({
          index,
          label: level.name || (level.height ? `${level.height}p` : `${Math.round(level.bitrate / 1000)} kbps`),
        })));
        if (resumeAt > 0 && resumeAt < video.duration * 0.95) video.currentTime = resumeAt;
        recoveryAttempts.current = { network: 0, media: 0 };
        setLoading(false);
        video.play().catch(() => undefined);
      });
      hls.on(Hls.Events.FRAG_LOADED, () => {
        recoveryAttempts.current = { network: 0, media: 0 };
      });
      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (!data.fatal) return;
        if (data.type === Hls.ErrorTypes.NETWORK_ERROR && recoveryAttempts.current.network < 2) {
          recoveryAttempts.current.network += 1;
          authApi.refresh().catch(() => undefined).finally(() => hls.startLoad());
          return;
        }
        if (data.type === Hls.ErrorTypes.MEDIA_ERROR && recoveryAttempts.current.media < 2) {
          recoveryAttempts.current.media += 1;
          hls.recoverMediaError();
          return;
        }
        setError("Playback could not recover. Please try again.");
        setLoading(false);
        hls.destroy();
        hlsRef.current = null;
      });
    };

    loadSource();
    return () => {
      active = false;
      hlsRef.current?.destroy();
      hlsRef.current = null;
    };
  }, [assetId, progressId, progressKind]);

  useEffect(() => {
    const timer = window.setInterval(persistProgress, 10_000);
    const onPageHide = () => persistProgress();
    window.addEventListener("pagehide", onPageHide);
    window.addEventListener("beforeunload", onPageHide);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("pagehide", onPageHide);
      window.removeEventListener("beforeunload", onPageHide);
      persistProgress();
    };
  }, [persistProgress]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      authApi.refresh().catch(() => undefined);
    }, 10 * 60_000);
    return () => window.clearInterval(timer);
  }, []);

  const togglePlay = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) video.play().catch(() => undefined);
    else video.pause();
  }, []);

  const toggleMute = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    video.muted = !video.muted;
    setMuted(video.muted);
  }, []);

  const toggleFullscreen = useCallback(() => {
    if (document.fullscreenElement) document.exitFullscreen().catch(() => undefined);
    else containerRef.current?.requestFullscreen().catch(() => undefined);
  }, []);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target?.matches("input, textarea, select")) return;
      const video = videoRef.current;
      if (!video) return;
      if (event.key === " ") {
        event.preventDefault();
        togglePlay();
      } else if (event.key === "ArrowLeft") video.currentTime = Math.max(0, video.currentTime - 10);
      else if (event.key === "ArrowRight") video.currentTime = Math.min(video.duration || 0, video.currentTime + 10);
      else if (event.key.toLowerCase() === "f") toggleFullscreen();
      else if (event.key.toLowerCase() === "m") toggleMute();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [toggleFullscreen, toggleMute, togglePlay]);

  const selectQuality = (nextQuality: number) => {
    setQuality(nextQuality);
    if (hlsRef.current) hlsRef.current.currentLevel = nextQuality;
  };

  return (
    <div ref={containerRef} className="group relative flex min-h-dvh w-full items-center justify-center overflow-hidden bg-black text-white">
      <video
        ref={videoRef}
        className="h-full max-h-dvh w-full object-contain"
        playsInline
        onClick={togglePlay}
        onPlay={() => setPlaying(true)}
        onPause={() => { setPlaying(false); persistProgress(); }}
        onTimeUpdate={(event) => setCurrentTime(event.currentTarget.currentTime)}
        onDurationChange={(event) => setDuration(event.currentTarget.duration || 0)}
        onVolumeChange={(event) => { setMuted(event.currentTarget.muted); setVolume(event.currentTarget.volume); }}
        onWaiting={() => setLoading(true)}
        onCanPlay={() => setLoading(false)}
      />

      <button type="button" onClick={() => router.back()} className="absolute left-3 top-3 z-20 flex min-h-11 min-w-11 items-center justify-center rounded-full bg-black/60 text-2xl hover:bg-black/90 sm:left-6 sm:top-6" aria-label="Back">←</button>
      {title && <h1 className="pointer-events-none absolute left-16 top-5 z-10 max-w-[65vw] truncate text-sm font-bold drop-shadow sm:left-24 sm:top-8 sm:text-lg">{title}</h1>}
      {loading && !error && <div className="pointer-events-none absolute inset-0 flex items-center justify-center"><span className="h-12 w-12 animate-spin rounded-full border-4 border-white/30 border-t-red-600" aria-label="Loading video" /></div>}
      {error && <div className="absolute inset-0 flex items-center justify-center bg-black/80 p-6 text-center"><div><p className="text-xl font-bold">{error}</p><button type="button" onClick={() => window.location.reload()} className="mt-5 rounded bg-white px-5 py-2 font-bold text-black">Retry</button></div></div>}

      <div className="absolute inset-x-0 bottom-0 z-20 bg-linear-to-t from-black via-black/75 to-transparent px-3 pb-4 pt-16 opacity-100 transition sm:px-6 sm:pb-6 md:opacity-0 md:group-hover:opacity-100 md:group-focus-within:opacity-100">
        <input aria-label="Seek" type="range" min={0} max={duration || 0} step="0.1" value={Math.min(currentTime, duration || 0)} onChange={(event) => { if (videoRef.current) videoRef.current.currentTime = Number(event.target.value); }} className="h-2 w-full accent-red-600" />
        <div className="mt-3 flex items-center gap-2 sm:gap-4">
          <button type="button" onClick={togglePlay} className="min-h-11 min-w-11 text-2xl" aria-label={playing ? "Pause" : "Play"}>{playing ? "❚❚" : "▶"}</button>
          <button type="button" onClick={toggleMute} className="min-h-11 min-w-11 text-xl" aria-label={muted ? "Unmute" : "Mute"}>{muted ? "🔇" : "🔊"}</button>
          <input aria-label="Volume" type="range" min={0} max={1} step={0.05} value={muted ? 0 : volume} onChange={(event) => { const next = Number(event.target.value); if (videoRef.current) { videoRef.current.volume = next; videoRef.current.muted = next === 0; } }} className="hidden w-24 accent-white sm:block" />
          <span className="whitespace-nowrap text-xs sm:text-sm">{timeLabel(currentTime)} / {timeLabel(duration)}</span>
          <div className="ml-auto flex items-center gap-2 sm:gap-4">
            <label className="flex min-h-11 items-center gap-2 text-xs sm:text-sm">Quality
              <select aria-label="Playback quality" value={quality} onChange={(event) => selectQuality(Number(event.target.value))} className="rounded border border-white/40 bg-black/80 px-2 py-2 text-white">
                <option value={-1}>Auto</option>
                {qualities.map((item) => <option key={item.index} value={item.index}>{item.label}</option>)}
              </select>
            </label>
            <button type="button" onClick={toggleFullscreen} className="min-h-11 min-w-11 text-xl" aria-label="Fullscreen">⛶</button>
          </div>
        </div>
      </div>
    </div>
  );
}
