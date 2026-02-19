# Hint 2: Check the ArgoCD Controller

The Application is a **custom resource**. For it to work, there needs to be an **operator** (controller) watching and acting on Application resources.

Check if the ArgoCD application controller is running:

```bash
kubectl get deployments -n argocd
kubectl get pods -n argocd | grep application-controller
```

What do you notice about the application controller? Is it running?

If the controller isn't running, Applications won't be processed - they'll just sit there as YAML in the cluster.