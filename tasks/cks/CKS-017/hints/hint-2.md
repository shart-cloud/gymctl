# CKS-017 — Hint 2 (cost 25)

Rewrite `app/Dockerfile` as two stages:

1. **Builder** — `FROM golang:1.22-alpine AS builder`. Copy `go.mod` and
   `main.go`, then `go build`.
2. **Final** — `FROM gcr.io/distroless/static-debian12`. `COPY --from=builder`
   the binary, set a non-root `USER`, and `ENTRYPOINT` the binary.

Three things that trip people up:

- **Static binary or bust.** distroless/static has *no libc*. If you build a
  normal dynamically-linked binary it will fail with "no such file or
  directory" at runtime. Build with `CGO_ENABLED=0` so the binary is fully
  static.
- **No shell in the final image.** distroless has no `/bin/sh`, so you can't use
  shell-form `CMD`, `RUN`, or exec into it to debug. Use exec-form
  `ENTRYPOINT ["/app"]`.
- **Drop root.** distroless ships a `nonroot` user (uid 65532). Add
  `USER nonroot` (or use the `:nonroot` tag) — otherwise the final image still
  runs as root and the non-root check fails.

Then build and verify:

```bash
docker build -t app:secure app
docker image inspect app:secure --format '{{.Size}}  user={{.Config.User}}'
```
