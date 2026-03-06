# Hint 2: NonIndexed vs Indexed Completion Mode

There are two completion modes for Jobs:

**NonIndexed** (the default):
- All pods are interchangeable — they all run the same command
- The Job tracks total successful completions, not which "items" finished
- Works when your app has its own coordination (e.g., pulls work from a queue)
- Pods have no built-in way to know their position or when to stop

**Indexed**:
- Each pod gets a unique index: 0, 1, 2 ... (completions-1)
- The index is available as the env var `JOB_COMPLETION_INDEX`
- The pod processes item N, exits 0, and the Job marks index N complete
- Works when your work is a fixed set of items (files, chunks, IDs)

Jerry's workers loop forever because they're NonIndexed — they have no index,
so they can't know what to process or when they're done.

Delete the broken job and recreate it with `completionMode: Indexed`:

```bash
kubectl delete job jerry-batch

# Edit the setup YAML or patch directly:
kubectl apply -f - <<'EOF'
apiVersion: batch/v1
kind: Job
metadata:
  name: jerry-batch
spec:
  completions: 6
  parallelism: 3
  completionMode: Indexed
  backoffLimit: 4
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: worker
        image: busybox:1.36
        command:
        - sh
        - -c
        - |
          echo "Processing item ${JOB_COMPLETION_INDEX}"
          sleep 5
          echo "Item ${JOB_COMPLETION_INDEX} done."
EOF
```
