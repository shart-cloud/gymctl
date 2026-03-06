# Hint 3: Apply the VPA Recommendation

Once `kubectl describe vpa jerry-app-vpa` shows a recommendation, read the Target:

```bash
kubectl get vpa jerry-app-vpa \
  -o jsonpath='{.status.recommendation.containerRecommendations[0].target}'
```

Then patch the deployment with those values (replace with actual output):

```bash
kubectl patch deployment jerry-app -p='{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "app",
          "resources": {
            "requests": {
              "cpu": "<target cpu from VPA>",
              "memory": "<target memory from VPA>"
            }
          }
        }]
      }
    }
  }
}'
```

Verify the rollout and confirm requests are lower than the original 800m/512Mi.
