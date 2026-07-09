# CKS-018 — Hint 1 (free)

Start by seeing what you actually ship. List the images in the namespace, then
scan each one, filtered to the severities you care about:

```bash
kubectl -n supply-chain get deploy \
  -o custom-columns=NAME:.metadata.name,IMAGE:.spec.template.spec.containers[0].image

trivy image --severity HIGH,CRITICAL nginx:1.20.0-alpine
```

`--severity HIGH,CRITICAL` cuts the noise down to what you must triage. The
number to read is the `Total:` line at the top of each table. You are hunting
for CRITICALs that have a **Fixed Version** — those are the ones a version bump
can close.
