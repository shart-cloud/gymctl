# CKS-020 — Solution

One object: a **ValidatingWebhookConfiguration** named `image-policy-webhook`
that connects the cluster's admission chain to the pre-built backend Service
`policy-system/image-policy`.

```bash
kubectl apply -f solution/
```

That is all the learner writes — the backend (Secret + ConfigMap + Deployment +
Service) is shipped in `setup/`. `kubectl apply -f solution/` alone takes the
broken state to 100% green; it is a pure add, no deletes.

## Why each field matters

| Field | Value | Why |
| --- | --- | --- |
| `clientConfig.service` | `policy-system/image-policy`, path `/validate`, port `8443` | Tells the API server *where* to send the AdmissionReview. Must match the Service name/namespace/port the backend listens on. |
| `clientConfig.caBundle` | CA of the fixed lab certs | The API server calls the webhook over HTTPS and verifies the backend's serving cert against this CA. If it does not match, the TLS handshake fails. |
| `rules` | `CREATE`,`UPDATE` on `pods` (`""`/`v1`) | Scopes *which* requests are sent to the webhook. The policy is about images, which can change on UPDATE too — so both operations are covered. |
| `namespaceSelector` | NotIn `kube-system`, `policy-system` | Carves out the control plane and the backend's own namespace (see deadlock note). |
| `failurePolicy` | `Fail` | Fail **closed**: if the backend is unreachable, deny rather than admit. The secure default. |
| `sideEffects` | `None` | The backend only reads and decides — no external mutations — so dry-run requests are safe to send. |
| `admissionReviewVersions` | `["v1"]` | The only AdmissionReview version the backend speaks. |

## The three ways this webhook can brick the cluster (and how the config avoids them)

1. **caBundle mismatch.** The API server dials the backend over TLS. If the
   `caBundle` is not the CA that signed the backend's `tls.crt`, the handshake
   fails → under `failurePolicy: Fail` **every** pod create in a non-excluded
   namespace is denied, including the trusted ones. The exercise ships fixed
   certs so the static caBundle always matches.

2. **Not excluding `policy-system` → backend deadlock.** If the webhook validated
   pods in its own backend's namespace and the backend were ever down (node
   reboot, rollout), the webhook would block the backend's replacement pod from
   starting — which keeps the webhook down — a self-inflicted, unrecoverable
   fail-closed loop. Excluding `policy-system` breaks the cycle.

3. **Not excluding `kube-system` → control-plane disruption.** System components
   create pods in `kube-system`; a fail-closed webhook that can't be reached
   would block them too. Exclude it.

The exclusion uses a `namespaceSelector` with a `matchExpressions` `NotIn` on the
built-in `kubernetes.io/metadata.name` label (auto-stamped on every namespace),
so no extra labelling is needed:

```yaml
namespaceSelector:
  matchExpressions:
    - key: kubernetes.io/metadata.name
      operator: NotIn
      values: ["kube-system", "policy-system"]
```

## Admission control is forward-looking

The `legacy-nginx` demonstrator (on `nginx:latest`) keeps running after you
install the webhook — admission only vets **new** requests, it does not evict
existing pods. That is expected. The checks prove enforcement with fresh
`--dry-run=server` creates, which invoke the webhook without persisting anything.

## Verify by hand

```bash
# rejected — Docker Hub + latest:
kubectl -n default run t1 --image=nginx:latest --dry-run=server
# rejected — Docker Hub, explicit tag:
kubectl -n default run t2 --image=nginx:1.27 --dry-run=server
# admitted — trusted registry, pinned tag:
kubectl -n default run t3 --image=registry.k8s.io/pause:3.9 --dry-run=server
# admitted — excluded namespace bypasses the webhook entirely:
kubectl -n policy-system run t4 --image=nginx:latest --dry-run=server
```
