# Hint 1: Understand What's Actually Happening

Start by looking at the Job status and what the pods are doing:

```bash
kubectl get job jerry-batch
kubectl get pods -l job-name=jerry-batch
kubectl describe job jerry-batch
```

You should see pods Running but `0/6` completions. Check what the pods are actually doing:

```bash
kubectl logs -l job-name=jerry-batch
```

A pod "completes" a Job item by exiting with code 0. What do you see in the logs?
Now look at the Job spec:

```bash
kubectl get job jerry-batch -o yaml | grep -E 'completions|parallelism|completionMode'
```

What is `completionMode` set to? What does that mean for how pods know which work item they own?
