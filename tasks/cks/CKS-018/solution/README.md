# CKS-018 — Solution

Roll all three Deployments to their patched, pinned tags:

```bash
kubectl apply -f solution/patched-workloads.yaml
# or, equivalently, one at a time:
kubectl -n supply-chain set image deploy/web-frontend  nginx=nginx:1.27.4-alpine
kubectl -n supply-chain set image deploy/session-cache redis=redis:7.4.2-alpine
kubectl -n supply-chain set image deploy/payments-app  app=python:3.12-alpine
```

Wait for the rollouts, then re-scan to confirm:

```bash
kubectl -n supply-chain rollout status deploy/web-frontend
kubectl -n supply-chain rollout status deploy/session-cache
kubectl -n supply-chain rollout status deploy/payments-app
trivy image --severity HIGH,CRITICAL nginx:1.27.4-alpine
```

## How to read a Trivy report

`trivy image --severity HIGH,CRITICAL <image>` prints one table per scan target
(the OS package layer, then each language layer). The columns that matter:

| Column | Meaning |
| --- | --- |
| `Library` | the vulnerable package (e.g. `openssl`, `libcrypto3`) |
| `Vulnerability` | the CVE id |
| `Severity` | Trivy's normalized severity — filter to `HIGH,CRITICAL` |
| `Installed Version` | what the image actually ships |
| `Fixed Version` | the version that closes the CVE — blank means no fix yet |

The line at the top of each table (`Total: N (HIGH: x, CRITICAL: y)`) is the
number you triage on.

### HIGH vs CRITICAL

- **CRITICAL** — remotely exploitable, typically no privileges or user
  interaction needed, high impact (RCE, auth bypass). Treat as *fix now*.
- **HIGH** — serious, but usually needs a precondition (local access, a specific
  configuration, or user interaction). Fix on the next planned bump.

In this exercise the gate is the CRITICAL offenders: any image whose scan shows
a CRITICAL with a `Fixed Version` available must be moved to a tag at or above
that fixed version. A CVE with an empty `Fixed Version` cannot be patched by a
version bump — you would mitigate it another way (drop the package, config, WAF).

## Why each image was bumped

| Workload | From | To | Why |
| --- | --- | --- | --- |
| `web-frontend` | `nginx:1.20.0-alpine` | `nginx:1.27.4-alpine` | 1.20.0 predates the fix for **CVE-2021-23017**, the nginx resolver off-by-one (CRITICAL). 1.27.x also refreshes the Alpine/OpenSSL layer that 1.20.0 pins to old, CVE-heavy versions. |
| `session-cache` | `redis:6.2.6-alpine` | `redis:7.4.2-alpine` | 6.2.6 carries several HIGH/CRITICAL Redis CVEs (Lua sandbox escapes, integer overflows) fixed across later 6.2.x / 7.x. 7.4.x is a supported line on a current Alpine base. |
| `payments-app` | `python:3.9-alpine` | `python:3.12-alpine` | 3.9's Alpine base and interpreter ship end-of-support CVEs. 3.12-alpine is a maintained interpreter on a patched base. |

## Keep it pinned

We moved to another **pinned** tag, not `:latest`. Pinning is what makes a scan
result reproducible and a rollout auditable — `:latest` silently drifts, so a
green scan today tells you nothing about what runs tomorrow. Patch by choosing a
newer pin, then re-scan that pin.

## Note on determinism

Trivy's vulnerability database updates continuously, so "re-scan shows zero
CRITICAL" is a moving target and is deliberately **not** the pass/fail gate here.
The gate is that each Deployment's running image equals the specific patched tag
above, plus its pod is Ready. Those are reproducible facts about the cluster;
the Trivy scan is the workflow that *tells you* which tag to move to.
