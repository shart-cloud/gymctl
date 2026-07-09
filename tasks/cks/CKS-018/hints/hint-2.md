# CKS-018 — Hint 2 (cost 25)

Scan all three and you'll find every one is on an outdated tag, but the fix is a
targeted bump of each Deployment to a newer **pinned** tag (not `:latest`):

| Deployment | Current (vulnerable) | Move to (patched) |
| --- | --- | --- |
| `web-frontend` | `nginx:1.20.0-alpine` | `nginx:1.27.4-alpine` |
| `session-cache` | `redis:6.2.6-alpine` | `redis:7.4.2-alpine` |
| `payments-app` | `python:3.9-alpine` | `python:3.12-alpine` |

`nginx:1.20.0` predates the fix for CVE-2021-23017 (the resolver CRITICAL); the
old redis and python bases carry their own HIGH/CRITICAL CVEs with fixes
available upstream.

Bump the image on a live Deployment with `kubectl set image`, or edit the
manifest and re-apply. Then `kubectl rollout status` each one — the new pods
have to pull the patched image and become Ready without breaking the app.

Traps:
- Keep it **pinned**. `:latest` would pass a scan today and drift tomorrow.
- Wait for the rollout to finish before you re-check; a half-rolled Deployment
  is neither fully patched nor Ready.
