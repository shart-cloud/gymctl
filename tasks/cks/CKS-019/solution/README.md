# CKS-019 — Solution

Auditing a Dockerfile is *read every instruction as an attacker would*. hadolint
finds the mechanical smells; the judgement calls (secrets, remote fetches,
privilege) are yours.

```bash
# In your work directory:
cd dockerfiles

# 1. Audit each original and read the DL codes.
for f in df-a df-b df-c; do
  echo "== $f =="
  docker run --rm -i hadolint/hadolint:v2.12.0 hadolint - < "$f"
done

# 2. Write the corrected file next to each original.
#    (These solution files are df-a.fixed / df-b.fixed / df-c.fixed.)

# 3. Build each fixed image to prove it still works.
for n in a b c; do
  docker build -f "df-$n.fixed" -t "cks019-df-$n:fixed" .
done
```

## The three files and what was wrong

### df-a — landing page

| Finding | Fix |
| --- | --- |
| `FROM alpine:latest` (DL3007) — floating tag, non-reproducible builds | Pin `FROM alpine:3.20` |
| `ADD https://…/README.md` — remote fetch at build time, non-reproducible, MITM foothold | `COPY index.html …` from the build context |
| Runs as root (no `USER`) | `adduser -D -u 10001 appuser` + `USER appuser` |
| No `HEALTHCHECK` | `HEALTHCHECK CMD ["true"]` |
| `CMD cat …` shell form (DL3025) | JSON exec form `CMD ["cat", …]` |

`ADD` of a URL is not flagged by hadolint — it is a *manual-review* finding. Use
`COPY` for local files and never `ADD` a URL; if you must fetch, do it with a
verified checksum in a build stage you can audit.

### df-b — worker

| Finding | Fix |
| --- | --- |
| `ENV API_KEY=sk_live_…` — secret baked into a layer, readable via `docker history` | Remove it; inject at runtime (`docker run -e API_KEY=…` or a mounted secret) |
| `RUN echo "DB_PASSWORD=hunter2" > …` — password written into a layer | Remove it; mount the secret at runtime |
| `wget … | sh` (DL4006) — runs unverified remote code | Remove the pipe-to-shell entirely; install from pinned, verified sources |
| `FROM alpine:latest` (DL3007) | Pin `FROM alpine:3.20` |
| Runs as root | `USER appuser` |

Deleting the secret from the *current* layer is not enough in a real image — a
leaked secret must be **rotated**, because it still lives in earlier build
history and anywhere the old image was pushed. Here we simply never introduce it.

### df-c — tools

| Finding | Fix |
| --- | --- |
| `FROM ubuntu:latest` (DL3007) | Pin `FROM debian:12-slim` (smaller, pinned) |
| `apt-get install sudo telnet netcat-traditional` — unneeded, risky packages (DL3008/DL3009/DL3015) | Drop them entirely; the image needs none of them |
| `chmod 777 /opt` — world-writable | `chmod 0755 /opt/app` (least privilege) |
| `sudo echo …` (DL3004) — sudo in a container | Remove; the build already runs as root |
| Runs as root, no `HEALTHCHECK` | `USER appuser` + `HEALTHCHECK CMD ["true"]` |

## Why these controls matter (CKS: Supply Chain Security)

- **Pinned tags** make builds reproducible and stop a silently-updated `:latest`
  from pulling in a compromised or breaking base.
- **No secrets in layers** — image layers are not private. Anything baked in is
  extractable by anyone who can pull the image.
- **No `curl|sh`** — piping a remote server's output into a shell executes
  whatever it sends, unverified. It is the textbook supply-chain injection point.
- **`COPY`, not remote `ADD`** — keep the build hermetic and auditable.
- **Non-root `USER`** — a container breakout as root is far worse than as UID
  10001. Drop privilege at build time so the default runtime user is unprivileged.

## Verify

```bash
cd dockerfiles
for n in a b c; do
  f="df-$n.fixed"
  grep -q ':latest' "$f" && echo "$f still floats latest" || echo "$f pinned OK"
  grep -q '^USER ' "$f" && echo "$f has USER" || echo "$f missing USER"
  docker run --rm -i hadolint/hadolint:v2.12.0 hadolint --no-fail - < "$f"   # expect no output
  docker build -f "$f" -t "cks019-df-$n:fixed" . >/dev/null && echo "$f builds"
done
```
