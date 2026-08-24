export const API_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type User = {
  id: string;
  name: string;
  email: string;
  role: "user" | "admin";
};

export type AuthResponse = { user: User };

export type CatalogItem = {
  id: number;
  title_id?: number;
  movie_id?: number;
  series_id?: number;
  asset_id?: string | null;
  title?: string;
  content_type: string;
  thumbnail_url: string;
  description?: string;
  year_released?: number;
  asset_status?: "pending" | "processing" | "ready" | "failed";
  published?: boolean;
  progress_kind?: ProgressKind;
  progress_id?: number;
  current_time_seconds?: number;
};

export type HomeRow = {
  id: number | string;
  title: string;
  items: CatalogItem[];
};

export type Episode = {
  id?: number;
  episode_id?: number;
  episode_number?: number;
  episode?: number;
  title: string;
  description: string;
  duration?: number | string;
  thumbnail_url?: string;
  asset_id?: string | null;
  asset_status?: "pending" | "processing" | "ready" | "failed";
};

export type Season = {
  id?: number;
  season_id?: number;
  season_number?: number;
  number?: number;
  episodes: Episode[];
};

export type TitleDetail = CatalogItem & {
  genres?: string[];
  categories?: string[];
  actors?: string[];
  cast?: string[];
  director?: string;
  duration?: number | string;
  seasons?: Season[];
};

export type Profile = {
  id: string;
  name: string;
  avatar: string | null;
  is_kids: boolean;
};

export type ProgressKind = "movie" | "episode";

export type WatchProgress = {
  current_time_seconds: number;
  updated_at?: string;
};

export type AdminAssetStatus = {
  status: "pending" | "processing" | "ready" | "failed" | "superseded";
  qualities: string[];
  error: string | null;
  duration?: number | null;
  source_width?: number | null;
  source_height?: number | null;
};

export type AdminTitleInput = {
  title: string;
  description: string;
  director: string;
  year_released: number;
  duration?: number;
  number_of_seasons?: number;
  genre_ids?: number[];
  actor_ids?: number[];
  category_ids?: number[];
};

export type CreatedTitle = {
  id: number;
  title_id: number;
  genre_ids?: number[];
  actor_ids?: number[];
  category_ids?: number[];
};

export type MetadataReference = {
  id: number;
  name: string;
};

export type Plan = {
  id: number;
  code: string;
  name: string;
  price: number;
  currency: string;
  quality: string;
  max_streams: number;
};

export type DiscountPreview = {
  valid: boolean;
  subtotal: number;
  discount_amount: number;
  total: number;
  currency: string;
  reason?: string;
};

export type Payment = {
  id: string;
  plan_id: number;
  method: "card" | "oxxo";
  status: "pending" | "paid" | "expired" | "failed";
  subtotal: number;
  discount_amount: number;
  total: number;
  amount: number;
  currency: string;
  reference?: string;
  card_last4?: string;
  card_brand?: string;
  expires_at?: string;
  paid_at?: string;
  simulated: boolean;
  created_at: string;
};

export type CardPaymentInput = {
  plan_id: number;
  code?: string;
  card: { number: string; exp: string; cvv: string; name: string };
};

type ErrorBody = { error?: string; message?: string };
type ApiOptions = RequestInit & { retryOnUnauthorized?: boolean };

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly body: ErrorBody | null,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

let refreshRequest: Promise<User> | null = null;

async function parseResponse<T>(response: Response): Promise<T> {
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

async function refreshSession(): Promise<User> {
  if (!refreshRequest) {
    refreshRequest = fetch(`${API_URL}/api/v1/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { Accept: "application/json" },
    })
      .then(async (response) => {
        if (!response.ok) {
          const body = (await response.json().catch(() => null)) as ErrorBody | null;
          throw new ApiError(
            body?.error ?? body?.message ?? "Your session has expired.",
            response.status,
            body,
          );
        }
        return (await parseResponse<AuthResponse>(response)).user;
      })
      .finally(() => {
        refreshRequest = null;
      });
  }
  return refreshRequest;
}

export async function apiRequest<T>(
  path: string,
  options: ApiOptions = {},
): Promise<T> {
  const { retryOnUnauthorized = true, headers, ...init } = options;
  const requestHeaders = new Headers(headers);
  requestHeaders.set("Accept", "application/json");
  if (init.body && !(init.body instanceof FormData)) {
    requestHeaders.set("Content-Type", "application/json");
  }

  const execute = () =>
    fetch(`${API_URL}${path}`, {
      ...init,
      headers: requestHeaders,
      credentials: "include",
    });

  let response = await execute();
  if (response.status === 401 && retryOnUnauthorized) {
    await refreshSession();
    response = await execute();
  }

  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as ErrorBody | null;
    throw new ApiError(
      body?.error ?? body?.message ?? `Request failed (${response.status})`,
      response.status,
      body,
    );
  }
  return parseResponse<T>(response);
}

const unwrapUser = (response: AuthResponse) => response.user;

export const authApi = {
  login: (email: string, password: string) =>
    apiRequest<AuthResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
      retryOnUnauthorized: false,
    }).then(unwrapUser),
  signup: (name: string, email: string, password: string) =>
    apiRequest<AuthResponse>("/api/v1/auth/signup", {
      method: "POST",
      body: JSON.stringify({ name, email, password }),
      retryOnUnauthorized: false,
    }).then(unwrapUser),
  me: () => apiRequest<User>("/api/v1/auth/me"),
  refresh: refreshSession,
  logout: () =>
    apiRequest<void>("/api/v1/auth/logout", {
      method: "POST",
      retryOnUnauthorized: false,
    }),
};

type ListResponse = CatalogItem[] | { items: CatalogItem[] } | { titles: CatalogItem[] };
function unwrapList(response: ListResponse): CatalogItem[] {
  if (Array.isArray(response)) return response;
  return "items" in response ? response.items : response.titles;
}

export const catalogApi = {
  home: () => apiRequest<HomeRow[]>("/api/v1/home"),
  titles: (query = "") =>
    apiRequest<ListResponse>(`/api/v1/titles${query}`).then(unwrapList),
  genres: () => apiRequest<MetadataReference[]>("/api/v1/genres"),
  actors: () => apiRequest<MetadataReference[]>("/api/v1/actors"),
  detail: async (item: CatalogItem) => {
    const type = item.content_type.toLowerCase();
    const series = type.includes("series") || type.includes("show");
    const resource = series ? "series" : "movies";
    const resourceId = series ? item.series_id ?? item.id : item.movie_id ?? item.id;
    try {
      return await apiRequest<TitleDetail>(`/api/v1/${resource}/${resourceId}`);
    } catch (reason) {
      if (!(reason instanceof ApiError) || reason.status !== 404) throw reason;
      return apiRequest<TitleDetail>(`/api/v1/titles/${item.title_id ?? item.id}`);
    }
  },
};

export type FavoriteItem = {
  title_id: number;
  title: string;
  content_type: string;
  thumbnail_url: string;
};

export type ContinueItem = {
  kind: ProgressKind;
  content_id: number;
  title_id: number;
  title: string;
  thumbnail_url: string;
  current_time_seconds: number;
  duration_seconds: number;
};

type FavoritesResponse = FavoriteItem[] | { favorites: FavoriteItem[] };
type ContinueResponse = ContinueItem[] | { items: ContinueItem[] };
type ProfilesResponse = Profile[] | { profiles: Profile[] };

export const userApi = {
  favorites: () =>
    apiRequest<FavoritesResponse>("/api/v1/favorites").then((response) =>
      Array.isArray(response) ? response : response.favorites,
    ),
  addFavorite: (titleId: number) =>
    apiRequest<void>("/api/v1/favorites", {
      method: "POST",
      body: JSON.stringify({ title_id: titleId }),
    }),
  removeFavorite: (titleId: number) =>
    apiRequest<void>(`/api/v1/favorites/${titleId}`, { method: "DELETE" }),
  continueWatching: () =>
    apiRequest<ContinueResponse>("/api/v1/progress/continue").then((response) =>
      Array.isArray(response) ? response : response.items,
    ),
  profiles: () =>
    apiRequest<ProfilesResponse>("/api/v1/profiles").then((response) =>
      Array.isArray(response) ? response : response.profiles,
    ),
  progress: (kind: ProgressKind, id: number) =>
    apiRequest<WatchProgress>(`/api/v1/progress/${kind}/${id}`),
  updateProgress: (kind: ProgressKind, id: number, currentTimeSeconds: number) =>
    apiRequest<WatchProgress>(`/api/v1/progress/${kind}/${id}`, {
      method: "PUT",
      body: JSON.stringify({ current_time_seconds: Math.max(0, Math.floor(currentTimeSeconds)) }),
    }),
};

export const paymentApi = {
  plans: () => apiRequest<Plan[]>("/api/v1/plans"),
  previewDiscount: (planId: number, code: string) =>
    apiRequest<DiscountPreview>("/api/v1/discounts/validate", {
      method: "POST",
      body: JSON.stringify({ plan_id: planId, code }),
    }),
  payByCard: (input: CardPaymentInput) =>
    apiRequest<Payment>("/api/v1/payments/card", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  createOxxo: (planId: number, code?: string) =>
    apiRequest<Payment>("/api/v1/payments/oxxo", {
      method: "POST",
      body: JSON.stringify({ plan_id: planId, ...(code ? { code } : {}) }),
    }),
  simulateOxxo: (reference: string) =>
    apiRequest<Payment>(`/api/v1/payments/oxxo/${encodeURIComponent(reference)}/simulate-payment`, {
      method: "POST",
    }),
  payment: (id: string) => apiRequest<Payment>(`/api/v1/payments/${encodeURIComponent(id)}`),
};

export const adminApi = {
  titles: () => catalogApi.titles("?limit=100&offset=0"),
  createTitle: (kind: "movies" | "series", input: AdminTitleInput) =>
    apiRequest<CreatedTitle>(`/api/v1/admin/${kind}`, { method: "POST", body: JSON.stringify(input) }),
  updateTitle: (kind: "movies" | "series", id: number, input: AdminTitleInput) =>
    apiRequest<CreatedTitle>(`/api/v1/admin/${kind}/${id}`, { method: "PATCH", body: JSON.stringify(input) }),
  publish: (titleId: number, published: boolean) =>
    apiRequest<{ published: boolean }>(`/api/v1/admin/titles/${titleId}/publish`, { method: "POST", body: JSON.stringify({ published }) }),
  assetStatus: (assetId: string) =>
    apiRequest<AdminAssetStatus>(`/api/v1/admin/assets/${assetId}`),
  createSeason: (seriesId: number, seasonNumber: number) =>
    apiRequest<{ id: number }>(`/api/v1/admin/series/${seriesId}/seasons`, {
      method: "POST",
      body: JSON.stringify({ season_number: seasonNumber, number_of_episodes: 0 }),
    }),
  createEpisode: (seasonId: number, episodeNumber: number, title: string) =>
    apiRequest<{ id: number }>(`/api/v1/admin/seasons/${seasonId}/episodes`, {
      method: "POST",
      body: JSON.stringify({ episode_number: episodeNumber, title, description: "", duration: 0 }),
    }),
};

export function artworkUrl(path: string): string {
  if (/^https?:\/\//i.test(path)) return path;
  if (path.startsWith("/api/")) return `${API_URL}${path}`;
  return path.startsWith("/") ? path : `${API_URL}/${path}`;
}

export function titleId(item: CatalogItem): number {
  return item.title_id ?? item.id;
}

export function isPlayable(item: CatalogItem): boolean {
  return Boolean(item.asset_id) && (!item.asset_status || item.asset_status === "ready");
}

export function watchHref(
  assetId: string,
  options: { kind?: ProgressKind; id?: number; title?: string } = {},
): string {
  const params = new URLSearchParams();
  if (options.kind && options.id) {
    params.set("kind", options.kind);
    params.set("id", String(options.id));
  }
  if (options.title) params.set("title", options.title);
  const query = params.toString();
  return `/watch/${encodeURIComponent(assetId)}${query ? `?${query}` : ""}`;
}
