# CKS-003 — Solution

Two independent fixes: put TLS in front of the dashboard, and strip its
ServiceAccount down from cluster-admin to read-only.

## Apply order (one caveat: `apply` cannot delete)

```bash
# 1. Everything additive lands in one shot:
kubectl apply -f solution/
#    - dashboard-tls Secret (kubernetes.io/tls)
#    - patched Ingress (tls block + force-ssl-redirect)
#    - dashboard-viewer ClusterRole + ClusterRoleBinding

# 2. Revoke cluster-admin. `kubectl apply` only creates/updates — it will NEVER
#    delete the old binding for you, so this step is manual and required:
kubectl delete clusterrolebinding dashboard-cluster-admin
```

That is the whole fix. Applying the directory again is safe (idempotent).

## Why each piece

| Object | Kind | Purpose |
| --- | --- | --- |
| `dashboard-tls` | Secret (`kubernetes.io/tls`) | Holds the cert + key for `dashboard.lab.local`. ingress-nginx reads it to terminate HTTPS. Its cert CN/SAN **is** `dashboard.lab.local` — a cert for another host would be wrong. |
| `dashboard` | Ingress | The `tls:` block binds the Secret to the host; the `force-ssl-redirect` annotation makes plain HTTP answer **308 → https** instead of serving cleartext. |
| `dashboard-viewer` | ClusterRole | Read-only (`get`/`list`/`watch`) on the resources a dashboard renders — pods, services, deployments, namespaces, nodes, etc. **No** secrets, **no** write verbs, **no** wildcards. |
| `dashboard-viewer` | ClusterRoleBinding | Grants that ClusterRole to the `dashboard` ServiceAccount. |
| `dashboard-cluster-admin` | ClusterRoleBinding | The vulnerability. **You delete this** — it is not in `solution/`. |

## The traps

1. **`apply` never deletes.** The `dashboard-cluster-admin` binding survives a
   `kubectl apply -f solution/`. Until you `kubectl delete` it, the SA still has
   cluster-admin *in addition* to the viewer role (RBAC is additive — the union
   of every binding wins). The negative checks stay red until it is gone.
2. **A tls block needs a matching Secret.** Reference a Secret that does not
   exist (or has the wrong host cert) and ingress-nginx silently falls back to
   its built-in *fake* certificate — HTTPS still answers 200, so a naive "does
   it serve 200" check would pass on the broken state. That is why the checks
   assert the redirect **and** the Secret's cert SAN.
3. **Redirect vs. TLS are two settings.** The `tls:` block enables HTTPS
   termination; the `force-ssl-redirect` annotation is what bounces HTTP → HTTPS.
   You need both — TLS available *and* cleartext refused.
4. **Least privilege means enumerate, don't wildcard.** `verbs: ["*"]` or
   `resources: ["*"]` would technically let the dashboard read, but it would also
   grant writes/secrets. Spell out read verbs on the specific resources.

## Verify by hand

```bash
SA=system:serviceaccount:kubernetes-dashboard:dashboard
CIP=$(kubectl -n ingress-nginx get svc ingress-nginx-controller -o jsonpath='{.spec.clusterIP}')

# TLS: HTTP redirects, HTTPS terminates (curl from inside the cluster —
# the gym's kind cluster has no host port mappings):
kubectl run t --rm -i --restart=Never --image=curlimages/curl:8.10.1 -- \
  -sS -o /dev/null -w 'http=%{http_code}\n'  --resolve dashboard.lab.local:80:$CIP  http://dashboard.lab.local/   # 308
kubectl run t --rm -i --restart=Never --image=curlimages/curl:8.10.1 -- \
  -sSk -o /dev/null -w 'https=%{http_code}\n' --resolve dashboard.lab.local:443:$CIP https://dashboard.lab.local/ # 200

# RBAC: viewer can read, cannot escalate, no longer cluster-admin:
kubectl auth can-i list pods                 --as=$SA   # yes
kubectl auth can-i get deployments -A        --as=$SA   # yes
kubectl auth can-i delete pods -A            --as=$SA   # no
kubectl auth can-i get secrets -A            --as=$SA   # no
kubectl auth can-i '*' '*' --all-namespaces  --as=$SA   # no
```
