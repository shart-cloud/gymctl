The current disruption policy is too strict for maintenance.

Check the deployment replica count and PDB side by side:

```bash
kubectl get deploy jerry-drain-app -o jsonpath='{.spec.replicas}'; echo
kubectl get pdb jerry-drain-pdb -o yaml
```

You need `disruptionsAllowed` to be at least `1`.
