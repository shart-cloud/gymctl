# CKS-017 — Solution

Replace `app/Dockerfile` with `solution/Dockerfile`, then build:

```bash
cp solution/Dockerfile app/Dockerfile
docker build -t app:secure app
```

## Why each piece is needed

| Piece | Purpose |
| --- | --- |
| `FROM golang:1.22-alpine AS builder` | A throwaway stage that holds the entire Go toolchain. It never ships. |
| `CGO_ENABLED=0 go build` | Produces a **static** binary with no libc dependency, so it can run on a base image that has no shared libraries at all. |
| `-ldflags "-s -w"` | Strips the symbol table and debug info — a smaller, quieter binary. |
| `FROM gcr.io/distroless/static-debian12` | The final base: no shell, no `apt`, no `curl`, no libc — just enough to run a static binary. Almost nothing to CVE or exec into. |
| `COPY --from=builder /app /app` | Copies **only** the compiled artifact out of the builder. The toolchain stays behind. |
| `USER nonroot` | distroless ships a `nonroot` user (uid 65532). The final image never runs as root. |
| `ENTRYPOINT ["/app"]` | Exec form — there is no shell in distroless to interpret shell-form commands. |

## The three traps

1. **Static or it won't start.** distroless/static has no libc. A default
   (dynamically linked) build dies at runtime with "no such file or directory".
   `CGO_ENABLED=0` makes the binary self-contained.
2. **No shell to fall back on.** You cannot `docker exec ... sh` into a
   distroless container, and shell-form `CMD`/`ENTRYPOINT` won't work. That's the
   point — an attacker who lands in the container has no shell either.
3. **Non-root is a separate decision.** A distroless base still defaults to root
   unless you say otherwise. Add `USER nonroot` (or use the `:nonroot` tag).

## The payoff

| | Bloated (Ubuntu, single stage) | Hardened (distroless, multi-stage) |
| --- | --- | --- |
| Size | ~870MB | ~7MB |
| Runs as | root | nonroot (65532) |
| Shell / package manager | yes (`/bin/sh`, `apt`) | none |
| Toolchain in image | yes (go, gcc, git) | none |

## Verify by hand

```bash
docker build -t app:secure app
docker image inspect app:secure --format 'size={{.Size}}  user={{.Config.User}}'
docker run -d -p 8080:8080 app:secure
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/   # -> 200
```
