type StatusPillProps = {
  status: string;
};

const colors: Record<string, string> = {
  uploading: "bg-blue-500/20 text-blue-200 ring-blue-500/40",
  pending: "bg-amber-500/20 text-amber-200 ring-amber-500/40",
  processing: "bg-amber-500/20 text-amber-200 ring-amber-500/40",
  ready: "bg-emerald-500/20 text-emerald-200 ring-emerald-500/40",
  published: "bg-emerald-500/20 text-emerald-200 ring-emerald-500/40",
  failed: "bg-red-500/20 text-red-200 ring-red-500/40",
  draft: "bg-zinc-500/20 text-zinc-200 ring-zinc-500/40",
};

export function StatusPill({ status }: StatusPillProps) {
  const normalized = status.toLowerCase();
  const label = normalized === "pending" ? "Processing" : normalized.charAt(0).toUpperCase() + normalized.slice(1);
  return <span className={`inline-flex shrink-0 whitespace-nowrap rounded-full px-3 py-1 text-xs font-bold ring-1 ${colors[normalized] ?? colors.draft}`}>{label}</span>;
}
