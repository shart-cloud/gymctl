# Hint 3: Scale the Controller Back Up

The ArgoCD application controller appears to be scaled to 0 replicas. This means no pod is watching Application resources.

Scale the controller back up:

```bash
kubectl scale deployment argocd-application-controller -n argocd --replicas=1

# Wait for it to be ready
kubectl rollout status deployment/argocd-application-controller -n argocd
```

Once the controller is running again, it will automatically notice the existing Application and start processing it.

Watch the Application status change:

```bash
kubectl get application jerry-simple-app -n argocd -w
```