# Hint 2: restartPolicy Must Be Never or OnFailure for Jobs

Jobs do not allow `restartPolicy: Always`. Kubernetes rejects it at admission.

But if the Job was already created with a different problem — the pod restarts — it means the `restartPolicy` is set in a way that causes the kubelet to restart on exit 0.

Valid values for a Job's pod template:

| restartPolicy | Behavior on success (exit 0) | Behavior on failure (exit non-0) |
|---|---|---|
| `Never` | Done — pod stays Completed, new pod created only on failure | New pod created |
| `OnFailure` | Done — pod stays Completed | Same pod restarted |
| `Always` | **Invalid for Jobs** | Kubernetes should reject this |

You need to delete and recreate the Job with the correct restartPolicy — you cannot edit this field in place.

```bash
kubectl delete job jerry-seed
```

Then create a new Job with `restartPolicy: Never`.
