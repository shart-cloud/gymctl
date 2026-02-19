Use container-specific logs:

```bash
pod=$(kubectl get pods -l app=jerry-log-mystery -o jsonpath='{.items[0].metadata.name}')
kubectl logs "$pod" -c log-shipper
kubectl logs "$pod" -c app
```

You need the logs from the failing container.
