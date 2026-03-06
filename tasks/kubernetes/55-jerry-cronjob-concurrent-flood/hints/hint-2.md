# Hint 2: Fix the CronJob Config

Edit the CronJob to prevent future floods:

```bash
kubectl edit cronjob jerry-reports
```

Change three things:

1. `concurrencyPolicy: Allow` → `concurrencyPolicy: Forbid`
2. `successfulJobsHistoryLimit: 100` → `successfulJobsHistoryLimit: 3`
3. `failedJobsHistoryLimit: 100` → `failedJobsHistoryLimit: 1`

Alternatively, patch it directly:

```bash
kubectl patch cronjob jerry-reports --type=merge -p='{
  "spec": {
    "concurrencyPolicy": "Forbid",
    "successfulJobsHistoryLimit": 3,
    "failedJobsHistoryLimit": 1
  }
}'
```

Changing the CronJob spec doesn't clean up existing Jobs — you need to do that separately.
