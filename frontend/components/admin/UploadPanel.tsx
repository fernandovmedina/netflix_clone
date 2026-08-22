"use client";

import { API_URL, adminApi, type AdminAssetStatus } from "@/utils/api/client";
import { useEffect, useRef, useState } from "react";
import { StatusPill } from "./StatusPill";

type UploadPanelProps = {
  endpoint: string;
  label: string;
  accept: string;
  pollAsset?: boolean;
  initialReadyAssetId?: string | null;
  onReady?: (assetId?: string) => void;
  onUploaded?: () => void;
};

type UploadResponse = { asset_id?: string; thumbnail_url?: string; status?: string; error?: string };

export function UploadPanel({ endpoint, label, accept, pollAsset = true, initialReadyAssetId, onReady, onUploaded }: UploadPanelProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [assetId, setAssetId] = useState<string | null>(initialReadyAssetId ?? null);
  const [asset, setAsset] = useState<AdminAssetStatus | null>(initialReadyAssetId ? { status: "ready", qualities: [], error: null } : null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!pollAsset || !assetId || asset?.status === "ready" || asset?.status === "failed") return;
    let active = true;
    let timer: number | undefined;
    const poll = async () => {
      try {
        const next = await adminApi.assetStatus(assetId);
        if (!active) return;
        setAsset(next);
        if (next.status === "ready") onReady?.(assetId);
        if (next.status !== "ready" && next.status !== "failed") timer = window.setTimeout(poll, 2_000);
      } catch (reason) {
        if (!active) return;
        setError(reason instanceof Error ? reason.message : "Unable to read processing status.");
        timer = window.setTimeout(poll, 4_000);
      }
    };
    poll();
    return () => {
      active = false;
      if (timer) window.clearTimeout(timer);
    };
  }, [asset?.status, assetId, onReady, pollAsset]);

  const upload = () => {
    const file = inputRef.current?.files?.[0];
    if (!file) {
      setError("Choose a file first.");
      return;
    }
    setUploading(true);
    setProgress(0);
    setError("");
    setAsset(null);
    const body = new FormData();
    body.append("file", file);
    const request = new XMLHttpRequest();
    request.open("POST", `${API_URL}${endpoint}`);
    request.withCredentials = true;
    request.upload.onprogress = (event) => {
      if (event.lengthComputable) setProgress(Math.round((event.loaded / event.total) * 100));
    };
    request.onload = () => {
      setUploading(false);
      let response: UploadResponse = {};
      try { response = JSON.parse(request.responseText) as UploadResponse; } catch { response = {}; }
      if (request.status < 200 || request.status >= 300) {
        setError(response.error ?? `Upload failed (${request.status}).`);
        return;
      }
      setProgress(100);
      onUploaded?.();
      if (pollAsset && response.asset_id) {
        setAssetId(response.asset_id);
        setAsset({ status: "pending", qualities: [], error: null });
      } else {
        onReady?.();
      }
    };
    request.onerror = () => {
      setUploading(false);
      setError("The upload connection failed.");
    };
    request.send(body);
  };

  const status = uploading ? "uploading" : asset?.status;

  return (
    <section className="rounded-xl border border-zinc-700 bg-zinc-900/70 p-4 sm:p-5">
      <div className="flex flex-wrap items-center justify-between gap-3"><h3 className="font-bold">{label}</h3>{status && <StatusPill status={status} />}</div>
      <div className="mt-4 flex flex-col gap-3 sm:flex-row">
        <input ref={inputRef} type="file" accept={accept} className="min-w-0 flex-1 rounded border border-zinc-600 bg-black p-2 text-base file:mr-3 file:rounded file:border-0 file:bg-white file:px-3 file:py-2 file:font-bold file:text-black" />
        <button type="button" disabled={uploading} onClick={upload} className="min-h-11 rounded bg-red-600 px-5 font-bold hover:bg-red-700 disabled:cursor-wait disabled:opacity-60">{uploading ? `Uploading ${progress}%` : "Upload"}</button>
      </div>
      {(uploading || progress > 0) && <div className="mt-3 h-2 overflow-hidden rounded bg-zinc-700"><div className="h-full bg-red-600 transition-all" style={{ width: `${progress}%` }} /></div>}
      {asset?.status === "processing" || asset?.status === "pending" ? <p className="mt-3 text-sm text-amber-200">The worker is creating adaptive renditions. This page updates automatically.</p> : null}
      {asset?.status === "ready" && <div className="mt-3 text-sm text-emerald-200"><p>Processing complete.</p>{asset.qualities.length > 0 && <p className="mt-1">Available qualities: {asset.qualities.join(" · ")}</p>}</div>}
      {asset?.status === "failed" && <p className="mt-3 rounded bg-red-950/60 p-3 text-sm text-red-200">Processing failed: {asset.error ?? "No worker error was provided."}</p>}
      {error && <p className="mt-3 text-sm text-red-300">{error}</p>}
    </section>
  );
}
