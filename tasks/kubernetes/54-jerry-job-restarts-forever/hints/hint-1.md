# Hint 1: Look at the Restart Count and restartPolicy

Check what's happening to the pod:

```bash
kubectl get pods -l job-name=jerry-seed
kubectl describe pod -l job-name=jerry-seed
```

Look at the **Restart Count** and **Reason** in the container status. The pod is exiting 0 (success) and restarting anyway.

Now check the Job spec:

```bash
kubectl get job jerry-seed -o yaml | grep restartPolicy
```

What value is set? What values are valid for a Job?
