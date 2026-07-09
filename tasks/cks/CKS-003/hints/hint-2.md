# CKS-003 — Hint 2 (cost 25)

## TLS half

1. Make a self-signed cert whose CN **and** SubjectAltName are
   `dashboard.lab.local` (the checks decode the cert and verify the host):

   ```bash
   openssl req -x509 -nodes -newkey rsa:2048 -keyout tls.key -out tls.crt \
     -days 3650 -subj "/CN=dashboard.lab.local/O=gymctl-lab" \
     -addext "subjectAltName=DNS:dashboard.lab.local"
   ```

2. Store it as a `kubernetes.io/tls` Secret named `dashboard-tls` in the
   `kubernetes-dashboard` namespace:

   ```bash
   kubectl create secret tls dashboard-tls -n kubernetes-dashboard \
     --cert=tls.crt --key=tls.key
   ```

3. Patch the Ingress: add a `tls:` block referencing `dashboard-tls` for host
   `dashboard.lab.local`, **and** the annotation
   `nginx.ingress.kubernetes.io/force-ssl-redirect: "true"` so plain HTTP
   answers `308 → https`. TLS termination and the redirect are two separate
   settings — you need both.

## RBAC half

1. **Delete** the ClusterRoleBinding that grants cluster-admin
   (`dashboard-cluster-admin`). `kubectl apply` cannot delete for you, and RBAC
   is additive — while that binding exists the SA is still cluster-admin no
   matter what else you add.
2. Create a `dashboard-viewer` ClusterRole with **only** `get`/`list`/`watch` on
   the resources a read-only dashboard shows (pods, services, deployments,
   namespaces, nodes, configmaps, …). No secrets, no write verbs, no wildcards.
3. Bind `dashboard-viewer` to the `dashboard` ServiceAccount with a new
   ClusterRoleBinding.

## Verify

```bash
SA=system:serviceaccount:kubernetes-dashboard:dashboard
kubectl auth can-i list pods     --as=$SA   # yes
kubectl auth can-i delete pods -A --as=$SA  # no
kubectl auth can-i get secrets -A --as=$SA  # no
```
