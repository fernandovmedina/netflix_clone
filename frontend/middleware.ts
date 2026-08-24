import { type NextRequest, NextResponse } from "next/server";

const AUTH_CHECK_TIMEOUT_MS = 1_500;

type SessionUser = {
  role?: unknown;
};

function redirectToLogin(request: NextRequest) {
  const { pathname, search } = request.nextUrl;
  const login = new URL("/login", request.url);
  login.searchParams.set("next", `${pathname}${search}`);
  return NextResponse.redirect(login);
}

async function getSessionUser(request: NextRequest): Promise<SessionUser | null> {
  const apiUrl = process.env.INTERNAL_API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), AUTH_CHECK_TIMEOUT_MS);

  try {
    const response = await fetch(`${apiUrl}/api/v1/auth/me`, {
      cache: "no-store",
      headers: {
        Accept: "application/json",
        Cookie: request.headers.get("cookie") ?? "",
      },
      signal: controller.signal,
    });

    if (!response.ok) return null;

    const user = (await response.json()) as SessionUser;
    return user.role === "admin" || user.role === "user" ? user : null;
  } catch {
    return null;
  } finally {
    clearTimeout(timeout);
  }
}

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  if (!request.cookies.get("access_token")?.value) {
    return redirectToLogin(request);
  }

  const user = await getSessionUser(request);
  if (!user) {
    return redirectToLogin(request);
  }

  if (pathname.startsWith("/admin") && user.role !== "admin") {
    return NextResponse.redirect(new URL("/home", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/home/:path*", "/admin/:path*", "/watch/:path*"],
};
