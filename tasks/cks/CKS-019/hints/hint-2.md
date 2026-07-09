# CKS-019 — Hint 2 (cost 25)

The categories you must remediate across the three files:

1. **Unpinned base** — every `FROM …:latest` becomes a specific tag
   (`alpine:3.20`, `debian:12-slim`, …). This is DL3007, and all three files have it.
2. **Non-root** — none of the originals set `USER`. Add a fixed-UID account and
   a `USER` line: `RUN adduser -D -u 10001 appuser` (alpine) /
   `RUN useradd --uid 10001 --create-home appuser` (debian), then `USER appuser`.
3. **Secrets (df-b)** — `ENV API_KEY=…` and the password written into `/app/db.env`
   both live in image layers forever. Delete them; inject at runtime instead.
4. **curl|sh (df-b)** — the `wget … | sh` line runs unverified remote code.
   Remove the pipe-to-shell entirely.
5. **Remote ADD (df-a)** — `ADD https://…` fetches at build time. Replace with
   `COPY` of the local file (`index.html` is already in the build context).
6. **Least privilege (df-c)** — drop the unneeded `sudo`/`telnet`/`netcat`
   install, replace `chmod 777` with `chmod 0755`, and remove the `sudo` line.

Traps:
- hadolint does **not** flag `ADD <url>` — df-a's remote fetch is a manual finding.
- Use JSON/exec form for `CMD`/`ENTRYPOINT` (`CMD ["cat","…"]`) or hadolint
  raises DL3025.
- A `.fixed` file that does not build is not a fix — build each one.
- Deleting a secret from the file is the minimum; a real leaked key must be rotated.
