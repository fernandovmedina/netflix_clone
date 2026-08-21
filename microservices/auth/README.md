# Auth Service

Stateless Go authentication gateway for the Netflix clone. It owns users,
password credentials, access tokens, refresh-token rotation, Google OAuth,
authorization gates, and authenticated reverse proxying.

## Authentication

- Passwords use bcrypt cost 12.
- Access tokens are HS256 JWT cookies valid for 15 minutes by default.
- Refresh tokens are opaque cookies valid for 30 days by default; only their
  SHA-256 hashes are stored. Every refresh rotates the token, and reuse revokes
  the entire session family.
- All instances share `JWT_SECRET` and PostgreSQL, so requests require neither
  sticky routing nor instance-local state.

## Endpoints

| Method | Path | Auth |
|---|---|---|
| POST | `/api/v1/auth/signup` | Public |
| POST | `/api/v1/auth/login` | Public |
| POST | `/api/v1/auth/refresh` | Refresh cookie |
| POST | `/api/v1/auth/logout` | Public; revokes cookie when present |
| GET | `/api/v1/auth/me` | Access token |
| GET | `/api/v1/auth/google` | Public |
| GET | `/api/v1/auth/google/callback` | OAuth state |
| ANY | `/api/v1/admin/*` | Admin access token |
| ANY | other owned `/api/v1/*` routes | Access token |

Access tokens may also be supplied as `Authorization: Bearer <token>`. The
gateway always removes incoming `X-User-Id`, `X-User-Email`, and `X-User-Role`
headers before injecting values from verified claims.

## Configuration

Required: `DATABASE_URL`, `JWT_SECRET` (at least 32 bytes), and downstream
service URLs. Optional settings include `ACCESS_TOKEN_TTL`,
`REFRESH_TOKEN_TTL`, `COOKIE_SECURE`, `CORS_ALLOWED_ORIGINS`, Google OAuth
credentials and redirect URL, `FRONTEND_URL`, and `PORT`.

For local HTTP, set `COOKIE_SECURE=false`. Set it to `true` behind HTTPS.
The migration runner bootstraps `ADMIN_EMAIL` / `ADMIN_PASSWORD`; its
`admin@netflix.local` / `admin12345` fallbacks are for local development only.

## Verify

```bash
go test ./...
go build ./...
go vet ./...
```
