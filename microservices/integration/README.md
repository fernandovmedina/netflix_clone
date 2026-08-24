# Integration tests

Start the complete local stack first:

```sh
docker compose up -d --build
```

Then run the end-to-end suite from this directory:

```sh
go test -tags=integration ./... -v
```

The tests always send user-facing requests through nginx. They read `BASE_URL`
(default `http://localhost:8080`), `DATABASE_URL` (default: the repository
`.env` value with the host rewritten to `localhost:5433`), `ADMIN_EMAIL`, and
`ADMIN_PASSWORD`. The latter two also default to the values in the repository
`.env` file.

The upload/transcode test reuses `seed/video/video.mp4` and requires the running
worker containers plus Docker CLI access. Skip that slower test with:

```sh
go test -tags=integration ./... -v -short
```
