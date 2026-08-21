# Codex 3 — Fixes from review round 1 (nginx)

Two confirmed findings from Codex 2. Scope: `microservices/nginx/nginx.conf` only.

---

### 1. `[MEDIUM]` No brute-force protection on login

Twelve consecutive wrong-password POSTs to `/api/v1/auth/login` all returned 401 with no throttling. Each one burns a cost-12 bcrypt on the server, so this is both a credential-guessing vector and a cheap CPU-exhaustion vector.

Fix: add a `limit_req_zone` keyed on `$binary_remote_addr` and apply it to the auth endpoints:
- `/api/v1/auth/login` and `/api/v1/auth/signup`: strict (a few requests/minute with a small burst).
- `/api/v1/auth/refresh`: looser — legitimate clients refresh routinely and a whole tab-full of requests can trigger one refresh each. Do not set this so tight that normal browsing logs people out.
- Return `429` (`limit_req_status 429`), and make sure the response still carries CORS headers or the browser will show an opaque failure instead of the real status.

Also add a `limit_conn` guard on the streaming routes so one client cannot open unlimited concurrent segment connections.

Do not rate-limit segment or manifest requests aggressively — a single player pulling an ABR stream legitimately issues many requests in a short window. Getting this wrong looks exactly like a broken player.

### 2. `[MEDIUM]` 5 GiB upload cannot actually succeed

`client_max_body_size 5g` rejects a request carrying a 5 GiB file, because multipart framing (boundaries, part headers, trailing CRLF) makes the encoded body larger than the file itself. A file at the documented maximum returns 413.

Fix: raise the nginx limit on the admin upload routes to `6g` so the transport layer is never the thing that rejects a legal file, and leave `MAX_UPLOAD_BYTES` as the single authoritative cap enforced in the application. The app must be what says no, with a clear JSON error — not nginx with an HTML 413.

Confirm `proxy_request_buffering off` on the upload location so nginx streams the body to the service instead of spooling 5 GiB to its own disk first.

---

## Definition of done

- `nginx -t` passes (you already have the stub-host docker technique for this).
- Demonstrate the rate limit actually returns 429 by exceeding it, and that a normal login still succeeds afterwards.
- Demonstrate a large upload is no longer rejected at the nginx layer (a `Content-Length` probe like Codex 2's is sufficient; you do not need to transfer 5 GiB).
- Report the results.

Do not commit. Do not touch Go files — Codex 1 is fixing the auth service concurrently.
