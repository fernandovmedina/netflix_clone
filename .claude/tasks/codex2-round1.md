# Codex 2 — Review round 1: authentication, migrations, Docker

Read `.claude/tasks/codex2-charter.md` first — it defines how you work and how you report. This file is the scope for **this round only**.

## Scope

Commits `130ff2e`, `c5632ac`, `b719bc1` on `main`. Concretely:

```
database/migrations/**      database/seed/**
microservices/shared/**     microservices/auth/**
docker-compose.yaml         .env.example    .gitignore
microservices/nginx/nginx.conf
microservices/*/Dockerfile
```

Use `git show c5632ac`, `git show b719bc1`, or just read the tree — everything in scope is committed.

`microservices/catalog`, `streaming`, `user` and `worker` contain **only a Dockerfile** right now; their Go source is being written in parallel as you review. Ignore them this round.

## What matters most in this round

This is the security foundation everything else sits on. Prioritise in this order:

1. **Token forgery and verification.** `microservices/shared/jwtutil`. Is the algorithm pinned to HS256, or will the parser accept `alg: none` or an RS256 token whose "signature" is the HMAC of the public key? Craft the malicious tokens and actually send them. Is `exp` enforced? Is `aud`/`iss` checked?
2. **Privilege escalation.** `microservices/auth/proxy.go` + `microservices/shared/authctx`. `authctx.Inject` claims to strip forged identity headers. Prove it: send `X-User-Role: admin` and `X-User-Id: <another uuid>` from a normal user's session and confirm neither survives to the downstream service. Also check header **case and encoding variants** (`x-user-role`, `X_User_Role`) and whether Go's canonicalisation covers them.
3. **Refresh-token rotation under concurrency.** `microservices/auth/repository.go`, `refresh_test.go`. Fire two simultaneous `/refresh` calls with the same token against two *different* auth replicas. Exactly one must succeed. Then replay a consumed token and confirm the whole family is revoked. This is where an atomic-UPDATE mistake hides.
4. **Multi-instance session continuity.** Log in through nginx, then curl auth1, auth2 and auth3 directly with the same cookies. All three must accept the session. Any instance-local state is CRITICAL.
5. **OAuth.** `microservices/auth/oauth.go`. Is `state` single-use, expiring, and bound to the PKCE verifier? Replay a consumed state. Can `redirect_to` be pointed at an external host (open redirect)? Is the Google `id_token` signature verified, or is the payload trusted as-is?
6. **SQL injection** across every query in `auth` and `database/seed`. Anything not parameterized.
7. **Migrations.** Run the runner twice concurrently against one database and confirm the advisory lock actually serialises it. Do any migrations `DROP` something that would destroy existing catalog data? Are the money columns `numeric`, never float?
8. **Secrets.** Is `.env` git-ignored and absent from history (`git log --all -- .env`)? Does `.env.example` contain any real value? Is the Google client secret or `JWT_SECRET` reachable from the frontend bundle or printed in any log line?
9. **Docker/nginx.** Containers running as root that need not. `client_max_body_size` adequate for 5 GiB uploads. Is the internal 8081 listener genuinely unpublished — can it be reached from the host, bypassing auth? **Try it:** `curl http://localhost:8081/catalog/...`. If a downstream tier is reachable without a token, that is CRITICAL.

## Also confirm

- No `supabase`, `gotrue`, or `jwks` reference remains under `microservices/` (`database/database.sql` and `exec.sql` are legacy reference files — those are expected and fine).
- `go test -race ./...` in `microservices/shared`, `microservices/auth`, `database/seed`.
- bcrypt cost is really 12 and the login path does not leak account existence by timing or by a different error message.

## Deliverable

The ranked finding list defined in the charter. Include the exact reproduction command and its real output for every CONFIRMED finding. Tag each with `Owner: codex1` or `Owner: codex3`.

Do not fix anything. Do not commit. Report only.

Work autonomously. Do not stop to ask for confirmation.
