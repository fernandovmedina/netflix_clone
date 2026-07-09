# Auth Service

Go microservice that is the single entry point of the backend. It handles
signup/login against Supabase Auth, validates the Supabase JWT on every
request, and redirects authenticated requests to the microservice that owns
the route (catalog, streaming, user). In production shape it runs as 5
containers behind an nginx load balancer (see `../docker-compose.yaml`).

## Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | — | Liveness check; returns the container name |
| POST | `/api/v1/auth/signup` | — | Body `{name, email, password}` → creates the user in Supabase Auth (name stored in user metadata) |
| POST | `/api/v1/auth/login` | — | Body `{email, password}` → Supabase session (`access_token`, `refresh_token`, `user`) |
| GET | `/api/v1/auth/user` | Bearer | Returns `{name, email, token}` of the logged-in user; 401 if the session is invalid/revoked |
| ANY | `/api/v1/*` | Bearer | Load-balancer path: JWT is verified, then the request is proxied to the owning service; 401 if not logged in, 503 if the service is not configured yet |

Authentication is a Supabase access token in the `Authorization: Bearer <jwt>`
header. Tokens are verified locally against the project's JWKS
(`/auth/v1/.well-known/jwks.json`); legacy projects can set
`SUPABASE_JWT_SECRET` to verify HS256 tokens instead.

## Route → service map (`proxy.go`)

- `titles`, `movies`, `series`, `serie`, `seasons`, `episodes`, `actors`, `categories`, `genres` → `CATALOG_SERVICE_URL`
- `stream` → `STREAMING_SERVICE_URL`
- `progress`, `favorites`, `profiles` → `USER_SERVICE_URL`

Proxied requests carry `X-User-Id` / `X-User-Email` headers so downstream
services don't re-parse the JWT.

## CORS

`cors.go` holds the manual allowlist of frontend origins (currently
`http://localhost:3000`). Add new origins to `allowedOrigins`.

## Environment (`.env.local`, git-ignored)

```
DATABASE_URL=            # Supabase Postgres (pgx pool)
SUPABASE_URL=            # https://<ref>.supabase.co
SUPABASE_ANON_KEY=       # publishable/anon key (same as the frontend)
SUPABASE_JWT_SECRET=     # optional — only for legacy HS256 projects
PORT=                    # optional, default 8080
CATALOG_SERVICE_URL=     # optional until the catalog service exists
STREAMING_SERVICE_URL=   # optional until the streaming service exists
USER_SERVICE_URL=        # optional until the user service exists
```

## Run

```bash
# single instance
go run .

# full load-balanced stack (5 containers + nginx on http://localhost:8080)
cd .. && docker compose up -d --build
```

Every request is logged with the container name (`[auth3] GET /api/v1/... -> 200`),
and nginx logs which upstream container served each request.
