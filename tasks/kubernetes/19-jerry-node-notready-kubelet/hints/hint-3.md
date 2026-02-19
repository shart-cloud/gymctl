Recover kubelet, then confirm scheduling:

```bash
NODE=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
docker exec "$NODE" systemctl start kubelet

kubectl wait --for=condition=Ready node/"$NODE" --timeout=120s
kubectl rollout restart deployment/jerry-node-app -n default
kubectl rollout status deployment/jerry-node-app -n default --timeout=120s
kubectl get pods -n default -l app=jerry-node-app -o wide
```
