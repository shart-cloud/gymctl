# CKS-017 — Hint 1 (free)

Start by seeing how big the problem is and what the app actually needs:

```bash
cat app/Dockerfile
docker build -t app:secure app        # ~870MB, and it runs as root
docker image inspect app:secure --format '{{.Size}} bytes, user=[{{.Config.User}}]'
```

The whole idea of the exercise: **the running app is a single static binary.**
Everything else in that Ubuntu image — the shell, `apt`, git, the C compiler,
the Go toolchain — is only needed to *build* it, never to *run* it. Ship the
binary, throw the rest away.

The one move that does it: a **multi-stage build**. Build in one stage, then
copy just the compiled binary into a tiny final stage.

Docs: <https://docs.docker.com/build/building/multi-stage/>
