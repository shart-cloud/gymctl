# Hint 3: Clean Up the Accumulated Jobs

List and delete all existing jobs created by jerry-reports:

```bash
# See them all
kubectl get jobs -l app=jerry-reports
# or by owner reference label that CronJobs set
kubectl get jobs

# Delete all jobs from this CronJob (and their pods)
kubectl delete jobs -l app=jerry-reports

# If the label selector doesn't match, delete by name pattern
kubectl get jobs --no-headers | awk '{print $1}' | grep jerry-reports | xargs kubectl delete job
```

Verify cleanup:

```bash
kubectl get jobs
kubectl get pods | grep jerry-reports
```

Both should show fewer than 5 entries total (only the most recent run should remain).
