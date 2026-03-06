# Hint 2: Install metrics-server for kind

VPA recommender requires metrics-server. Install it with the kind-compatible patch:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

kubectl -n kube-system patch deployment metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'

kubectl -n kube-system rollout status deployment/metrics-server
```

Wait ~2 minutes for VPA recommender to gather data, then:

```bash
kubectl describe vpa jerry-app-vpa
```

Watch for `Status.Recommendation` to populate.
