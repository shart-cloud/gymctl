# Hint 3: Watch It Complete

After recreating the Job with `completionMode: Indexed`, watch it run:

```bash
kubectl get pods -l job-name=jerry-batch -w
```

You should see pods named `jerry-batch-0-xxxxx`, `jerry-batch-1-xxxxx`, etc.
Each pod runs, exits 0, and a new batch starts until all 6 are done.

Check progress:

```bash
kubectl get job jerry-batch
# COMPLETIONS column should tick up: 0/6, 1/6, 2/6 ... 6/6

kubectl describe job jerry-batch | grep -A5 "Conditions"
# Should eventually show: Complete True
```

Verify the logs show index-aware work:

```bash
kubectl logs -l job-name=jerry-batch --prefix
# Should show: Processing item 0, Processing item 1, etc.
```

Once `COMPLETIONS` hits `6/6` and `SUCCESSFUL` matches, the Job is done.
