Try a drain and inspect why it is blocked:

```bash
node=$(kubectl get nodes -l jerry.gym/drain-target=true -o jsonpath='{.items[0].metadata.name}')
kubectl drain "$node" --ignore-daemonsets --delete-emptydir-data --timeout=60s
kubectl describe pdb jerry-drain-pdb
```
