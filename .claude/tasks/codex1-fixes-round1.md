# Codex 1 — Fixes from review round 1

Codex 2 reviewed the auth service and confirmed these defects with live reproductions. Fix all of them. They are small; do them carefully rather than quickly.

Scope: `microservices/auth/**`, `microservices/shared/authctx/**`. Add a regression test for **each** fix — a fix without a test that fails before it does not count.

---

### 1. `[MEDIUM]` Login timing leak enumerates OAuth-only accounts — `handlers.go:83`

Measured: a nonexistent account takes ~0.260 s (dummy bcrypt), a password-backed account ~0.260 s, but an **OAuth-only** account (`password_hash IS NULL`) returns in ~0.0018 s because the code short-circuits before bcrypt. That 140× gap reliably identifies which emails are registered Google users.

Fix: when the user is missing **or** has no password hash, still compare against the cost-12 dummy hash before returning the generic error. All three paths must do one bcrypt comparison.

Test: assert the three paths are within a sane ratio of each other (compare against the dummy-hash path; do not assert wall-clock absolutes, which are flaky in CI — assert that the no-password path actually invokes the comparison).

### 2. `[MEDIUM]` Signup accepts malformed emails — `handlers.go:35`

`{"email":"not-an-email"}` currently returns 201 with a live session.

Fix: validate with `net/mail.ParseAddress`, require exactly one mailbox, reject a display-name form (`Foo <a@b.c>` must not be accepted as the identifier — store the bare address), normalise to lowercase, and 400 on failure. Apply to login too, so the two agree on what an identifier is.

### 3. `[MEDIUM]` Passwords over 72 bytes return 500 — `handlers.go:39`

bcrypt errors out past 72 bytes and it surfaces as `{"error":"internal server error"}`.

Fix: validate password length in **bytes** (not runes) as 8–72 before hashing, return 400 with a clear message. Check the same bound on any password-change path. Note the distinction: a 30-character password of multi-byte emoji exceeds 72 bytes — the error must say so rather than 500.

### 4. `[MEDIUM]` `X_User_Role` underscore variant survives the proxy — `shared/authctx/context.go`

`authctx.Strip` deletes the three canonical `X-User-*` headers, but a client-sent `X_User_Role: admin` passes straight through to the downstream service alongside the correct injected header.

It does not escalate today only because our Go services read the hyphenated name. That is luck, not a control: several proxies and CGI-style layers normalise `_` to `-`, and adding one in front of a service would silently turn this into privilege escalation.

Fix: in `Strip`, delete **any** header whose canonical form matches `x-user-*` with `-` and `_` treated as equivalent — iterate `r.Header` and drop every key that normalises to one of the three identity names. Do not just add three more `Del` calls for the underscore spellings; a mixed variant like `X-User_Role` must die too.

Test: send `X-User-Role`, `x-user-role`, `X_User_Role`, `X-User_Role` and `x_user_id` in one request and assert none survives and only the signed values are injected.

---

## Definition of done

- `go build ./... && go vet ./... && go test -race ./...` clean in every module you touched.
- Each of the four has a test that fails without the fix.
- Report what you changed and paste the test output.

Do not commit. Do not touch files outside the scope above — Codex 3 is working in `frontend/` and `nginx/` concurrently.
