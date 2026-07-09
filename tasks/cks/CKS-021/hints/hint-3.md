# CKS-021 — Hint 3 (cost 50)

Here is the hardened `web` workload. The `api` one is the same shape without the
host access and without the emptyDir mounts (a `sleep` needs nothing writable).

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: staging
  labels: { app: web }
spec:
  replicas: 1
  selector: { matchLabels: { app: web } }
  template:
    metadata: { labels: { app: web } }
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 101
        seccompProfile: { type: RuntimeDefault }
      containers:
        - name: nginx
          image: nginxinc/nginx-unprivileged:1.27-alpine
          ports: [ { containerPort: 8080 } ]
          securityContext:
            runAsNonRoot: true
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            privileged: false
            capabilities: { drop: [ ALL ] }
          resources:
            requests: { cpu: 25m, memory: 32Mi }
            limits: { cpu: 100m, memory: 64Mi }
          volumeMounts:
            - { name: tmp,   mountPath: /tmp }
            - { name: cache, mountPath: /var/cache/nginx }
            - { name: run,   mountPath: /var/run }
      volumes:
        - { name: tmp,   emptyDir: {} }
        - { name: cache, emptyDir: {} }
        - { name: run,   emptyDir: {} }
```

For `api`: drop `hostNetwork` / `hostPID` / the `hostPath` volume entirely, use
`busybox:1.36` as `runAsUser: 1000`, set `readOnlyRootFilesystem: true`, drop
ALL caps, `allowPrivilegeEscalation: false`, and add cpu/memory
requests + limits — no emptyDir needed.

The full, ready-to-apply manifests are in `solution/`:

```bash
kubectl apply -f solution/
```
