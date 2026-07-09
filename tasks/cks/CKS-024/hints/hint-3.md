# CKS-024 — Hint 3 (cost 50)

Per-deployment pod template (repeat for web, api, cache):

```yaml
    spec:
      containers:
        - name: web            # api / cache
          image: busybox:1.36
          command: ["sh", "-c", "sleep 36000"]
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
          volumeMounts:
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: tmp
          emptyDir: {}
```

Then:

```bash
kubectl apply -f solution/workloads.yaml
```

Full manifest for all three in `solution/workloads.yaml`.
