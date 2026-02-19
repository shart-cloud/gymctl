Patch the PDB to allow a controlled disruption, then retry maintenance:

```bash
kubectl patch pdb jerry-drain-pdb --type merge -p '{"spec":{"minAvailable":2}}'

node=$(kubectl get nodes -l jerry.gym/drain-target=true -o jsonpath='{.items[0].metadata.name}')
kubectl drain "$node" --ignore-daemonsets --delete-emptydir-data --timeout=180s
kubectl uncordon "$node"
kubectl rollout status deployment/jerry-drain-app --timeout=180s
```
