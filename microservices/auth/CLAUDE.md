# Netflix Clone — Auth Service

The binding system contract is `../../docs/ARCHITECTURE.md`.

This service is the stateless API gateway and authentication owner. It uses
PostgreSQL for users, OAuth states, and refresh sessions; HS256 access tokens
are verified locally with the shared `JWT_SECRET`. Never trust client-provided
`X-User-*` headers: remove them and inject only signed claim values.

Keep refresh rotation atomic, store only token hashes, retain per-container
request logging, and keep the CORS allowlist environment-driven. Shared JWT,
database, JSON, identity-header, storage, and rendition helpers live in
`../shared`.
