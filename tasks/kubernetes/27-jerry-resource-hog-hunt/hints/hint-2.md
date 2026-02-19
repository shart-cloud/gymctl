Compare the two deployments:

```bash
kubectl get deploy
kubectl top pods | sort -k2 -h
kubectl describe deploy jerry-hog
```

The fix is in the Deployment spec, not in cluster-level settings.
