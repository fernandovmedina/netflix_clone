# Codex 3 — Fixes from review round 3

Codex 2's final security review found **no critical or high** issues. Two are yours, both confirmed with live reproductions. Do these after finishing round 4.

---

## 1. `[MEDIUM]` Any nonempty `access_token` cookie bypasses the frontend route guard
`frontend/middleware.ts:3`

```
/admin  no cookie                              → 307 → /login?next=%2Fadmin
/admin  Cookie: access_token=attacker-garbage  → 200, 14658 bytes
/home   Cookie: access_token=attacker-garbage  → 200, 14107 bytes
```

The middleware treats "a cookie exists" as "the user is authenticated", so a garbage value opens the protected page shells — including `/admin`, which a *normal* user's real cookie also opens. The backend APIs stay protected, so no admin data leaks; what leaks is the admin UI shell, and it is a bad guard regardless.

Fix: validate the cookie by calling `/api/v1/auth/me` from the middleware, require `role === "admin"` for `/admin`, and redirect on anything that is not a 200 — including timeouts and API failures. **Fail closed.** Keep the existing `?next=` redirect behaviour.

Watch the cost: this puts a network call in the middleware path. Keep it to the protected routes, use a short timeout, and do not block static assets.

## 2. `[MEDIUM]` No rate limit on discount validation or OXXO simulation
`microservices/nginx/nginx.conf:110`

```
30 rapid requests:
  login     → 1×401, 29×429   (protected)
  discount  → 30×200          (unprotected)
  oxxo      → 30×404          (unprotected)
```

An authenticated user can enumerate discount codes as fast as they can send requests, and generate unlimited payment-simulation traffic. Add dedicated `limit_req` zones for `POST /api/v1/discounts/validate` and the OXXO simulate route, sized so a normal checkout is never throttled — a person applying a code types it once, a script tries thousands.

Note the limitation honestly in your report: `limit_req_zone $binary_remote_addr` keys on client IP, which does not stop a distributed attacker. If you think a per-user server-side limit is warranted in the user service, say so and I will task Codex 1 — do not implement it yourself.

## 3. `[LOW]` — already fixed, just confirm

`pnpm lint` was failing with 150 errors from `frontend/public/hls.min.js`, a scratch file I left behind. I have deleted it along with `hlstest.html` and `rwd.html`. Confirm `pnpm lint` is clean now; if the harness in round 4 is still useful to you, keep your own copy outside `frontend/public/` so it never gets linted or shipped.

---

## Definition of done

- `pnpm build`, `pnpm lint`, `pnpm exec tsc --noEmit` all clean.
- Both reproductions above re-run and pasted with corrected output — in particular `/admin` with a garbage cookie, with an expired cookie, **and** with a valid normal-user cookie, all of which must redirect.
- Confirm you did not slow down unauthenticated page loads measurably.

Do not commit. Do not touch the Go services.

Work autonomously; do not stop to ask for confirmation.
