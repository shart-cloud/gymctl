# CKS-019 — Hint 3 (cost 50)

Here is `df-a.fixed` in full. Apply the same discipline to `df-b` (strip the
secrets and the `wget|sh`) and `df-c` (drop `sudo`/junk packages, `chmod 0755`,
add `USER` + `HEALTHCHECK`).

```dockerfile
# df-a.fixed
FROM alpine:3.20

# COPY the local file instead of ADDing a remote URL.
COPY index.html /var/www/index.html

# Non-root, fixed UID.
RUN adduser -D -u 10001 appuser
USER appuser

HEALTHCHECK CMD ["true"]

CMD ["cat", "/var/www/index.html"]
```

Verify one file end to end:

```bash
cd dockerfiles
docker run --rm -i hadolint/hadolint:v2.12.0 hadolint --no-fail - < df-a.fixed   # no output = clean
docker build -f df-a.fixed -t cks019-df-a:fixed .                                 # exits 0
```

The complete `df-b.fixed` and `df-c.fixed` are in `solution/`.
