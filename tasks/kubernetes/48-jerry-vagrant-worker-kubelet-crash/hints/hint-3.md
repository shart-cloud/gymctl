If your local shell is missing cluster access, export kubeconfig:

```bash
eval "$(gymctl kubeconfig --format env)"
kubectl get nodes -o wide
```

Then re-run checks:

```bash
gymctl check
```
