import type { TitleDetail } from "@/utils/api/client";
import { headers } from "next/headers";
import { cache } from "react";

export type AdminEditorKind = "movies" | "series";

export const getAdminTitle = cache(async (kind: AdminEditorKind, id: number): Promise<TitleDetail | null> => {
  if (!Number.isInteger(id) || id < 1) return null;
  const requestHeaders = await headers();
  const cookie = requestHeaders.get("cookie");
  const apiUrl = process.env.INTERNAL_API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  try {
    const response = await fetch(`${apiUrl}/api/v1/${kind}/${id}`, {
      cache: "no-store",
      headers: cookie ? { cookie } : undefined,
    });
    if (!response.ok) return null;
    return await response.json() as TitleDetail;
  } catch {
    return null;
  }
});
