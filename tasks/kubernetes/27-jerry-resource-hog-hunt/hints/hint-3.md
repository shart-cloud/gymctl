Patch resources on the noisy deployment:

```bash
kubectl set resources deployment/jerry-hog \
  --requests=cpu=150m,memory=128Mi \
  --limits=cpu=300m,memory=256Mi

kubectl rollout status deployment/jerry-hog
kubectl get deploy jerry-hog -o yaml | grep -A8 resources:
```

Make sure both requests and limits are set.
