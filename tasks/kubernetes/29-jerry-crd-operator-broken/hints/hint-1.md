# Hint 1: Check the Application Status

The Application resource exists but something is wrong with how it's being processed.

Check the Application status and events:

```bash
kubectl get application jerry-simple-app -n argocd
kubectl describe application jerry-simple-app -n argocd
```

Look at the **status** section and any **Events**. What does the sync status show?

Also check if there are any conditions or error messages in the Application's status.