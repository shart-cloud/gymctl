# Hint 3: Install metrics-server for kind

Install metrics-server and configure it for kind clusters:

```bash
# Install metrics-server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Patch for kind - allow insecure TLS
kubectl -n kube-system patch deployment metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'

# Wait for it to start
kubectl -n kube-system rollout status deployment/metrics-server
```

After ~60 seconds, verify metrics are flowing:

```bash
kubectl top nodes
kubectl top pods
```

The HPA will automatically start working once it can read CPU metrics.