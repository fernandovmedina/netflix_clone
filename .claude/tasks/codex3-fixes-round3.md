# Codex 3 — Round 3 fix: nginx caches upstream IPs, breaking auth after any restart

Your session was restarted, so assume no memory of earlier rounds. Background: `docs/ARCHITECTURE.md` (§2 service topology), `INTEGRATION.md`. Scope for this task: `microservices/nginx/**` and `docker-compose.yaml` only. Do not touch `frontend/**` or any Go service.

## The bug (found by the lead, reproduced live)

`microservices/nginx/nginx.conf` declares upstreams by container hostname:

```
upstream auth_service { least_conn; server auth1:8080; server auth2:8080; server auth3:8080; }
```

OSS nginx resolves those names **once at worker startup** and caches the IPs forever. Docker reassigns container IPs on restart, so as soon as any container restarts in a different order — which `restart: unless-stopped` guarantees after a host reboot — nginx keeps proxying to IPs that now belong to **a different service**.

Live reproduction on the running stack, before the fix:

```
$ for i in $(seq 1 6); do curl -s -o /dev/null -w '%{http_code} ' -b user.cookies \
    http://localhost:8080/api/v1/auth/me; done
404 200 200 404 200 200

$ docker compose logs nginx | grep auth/me
172.22.0.1 -> 172.22.0.8:8080  "GET /api/v1/auth/me HTTP/1.1" 404      <-- streaming1
172.22.0.1 -> 172.22.0.2:8080  "GET /api/v1/auth/me HTTP/1.1" 200      <-- auth2
172.22.0.1 -> 172.22.0.11:8080 "GET /api/v1/auth/me HTTP/1.1" 200      <-- auth3

$ docker inspect ... # actual IPs at that moment
172.22.0.6  auth1      172.22.0.8  streaming1
```

nginx had `auth1` pinned to `172.22.0.8`, which by then was **streaming1**. One third of every authenticated API request was being handed to the streaming service, which 404s it. `docker compose restart nginx` clears it (all 9 requests then returned 200) — but that is a manual workaround, not a fix, and it silently returns the next time anything restarts.

Severity: this breaks authentication, admin routes, catalog and streaming for a fraction of all traffic, with no error anywhere except the status code. It also means `docker compose restart auth2` — or scaling instances up and down, which is the whole point of the architecture — corrupts routing.

## What to fix

Make upstream resolution dynamic so nginx survives container restarts and instance count changes without a config edit or a restart.

Pick the approach you can prove works with **OSS nginx** (no nginx-plus-only directives such as `resolve` on a `server` line, and no commercial modules). The two workable shapes:

1. **Docker DNS + variable `proxy_pass`.** `resolver 127.0.0.11 valid=5s ipv6=off;` plus a `set $backend ...;` and `proxy_pass http://$backend;` per location, so the name is re-resolved at request time. Note the two traps: a variable `proxy_pass` drops the URI unless you restate it (`proxy_pass http://$backend$request_uri;` or a `rewrite ... break`), and it bypasses the `upstream` block, so `least_conn` is replaced by whatever order Docker's DNS returns.

2. **Compose replicas + DNS round-robin.** Collapse `auth1/auth2/auth3` into a single `auth` service with `deploy: {replicas: 3}` — Docker's embedded DNS then returns all replica IPs for the name `auth`, so the same variable-`proxy_pass` approach load-balances across whatever is currently running, and `docker compose up -d --scale auth=5` becomes the entire scaling procedure. Same for `catalog`, `streaming`, `user` and `worker`.

Option 2 is the better architecture and matches the project's stated scaling goal (today, adding an instance requires editing both compose and nginx.conf — a documented manual step that this would delete). **Take option 2 if you can make it work cleanly**, including:

- the per-instance log prefix (`[auth2] GET ... -> 200`) still identifying which container served — it can become the container hostname/id rather than a hard-coded name, but the lead must still be able to see the distribution across instances;
- healthchecks and `depends_on` still gating startup correctly;
- the internal `:8081` listener (catalog/streaming/user tiers) getting the same treatment — it has the identical bug;
- `migrate` and `seed` one-shot jobs still ordering correctly ahead of the services.

If replicas turn out to break something you cannot resolve, fall back to option 1 and say why in your report.

Keep everything else about the config: the rate-limit zones, the CORS origin map, `client_max_body_size` overrides for admin uploads, `proxy_request_buffering off`, the stream connection guard, Range/If-Range passthrough for segments, and the `@rate_limited` JSON error page. Do not weaken any of them while restructuring.

## Prove it

Do not report done on a config read-through. Run all of these against the live stack and paste the output:

1. Baseline: 12 authenticated requests, all 200, and the log distribution showing every instance serving some of them.
2. `docker compose restart <one auth instance>`, wait for healthy, then repeat step 1 — still all 200, and the restarted container is back in rotation under its **new** IP.
3. Restart a catalog, a streaming and a user instance and re-run the equivalent request for each tier (`/api/v1/titles`, a `master.m3u8` fetch, a watch-progress read) — all 200 with no stale-IP 404s.
4. Scale up and back down (`--scale auth=5` then `=3`, or add/remove an instance) with no nginx.conf edit and no nginx restart, and show requests still succeeding throughout.
5. Confirm a segment fetch still returns `206` with `Content-Range` and `Cache-Control: ...immutable`, and that an admin multipart upload larger than 1 MiB is still accepted (the `client_max_body_size` override survived).

Use a normal user session cookie for the authenticated calls; create one with `POST /api/v1/auth/signup` if you need it. The stack is already running — `docker compose ps`. Codex 2 is running a security review against the same stack concurrently, so **announce in your report any moment you restarted or recreated services**, and avoid `docker compose down`.

## Definition of done

- Restarting any single service, in any order, never produces a misrouted request.
- Adding or removing an instance requires no nginx.conf change.
- All five proofs above pasted with real output.
- `docs/ARCHITECTURE.md` §2 and the scaling note updated to describe the new procedure if you changed the topology.

Do not commit. Report what you changed, what you proved, and anything still broken.

Work autonomously; do not stop to ask for confirmation.
