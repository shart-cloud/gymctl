# CKS-018 — Hint 3 (cost 50)

Patch all three, then wait for the rollouts:

```bash
kubectl -n supply-chain set image deploy/web-frontend  nginx=nginx:1.27.4-alpine
kubectl -n supply-chain set image deploy/session-cache redis=redis:7.4.2-alpine
kubectl -n supply-chain set image deploy/payments-app  app=python:3.12-alpine

kubectl -n supply-chain rollout status deploy/web-frontend
kubectl -n supply-chain rollout status deploy/session-cache
kubectl -n supply-chain rollout status deploy/payments-app
```

Or apply the ready-made manifest, which does the same thing declaratively:

```bash
kubectl apply -f solution/patched-workloads.yaml
```

Confirm the running images and that each is Ready:

```bash
kubectl -n supply-chain get deploy \
  -o custom-columns=NAME:.metadata.name,IMAGE:.spec.template.spec.containers[0].image,READY:.status.readyReplicas
```

The full patched manifest and a walkthrough of reading a Trivy report are in
`solution/` (`patched-workloads.yaml` + `README.md`).
