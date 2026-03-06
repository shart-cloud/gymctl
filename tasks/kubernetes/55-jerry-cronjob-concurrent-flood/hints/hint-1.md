# Hint 1: Assess the Damage

First, understand what you're dealing with:

```bash
kubectl get cronjob jerry-reports
kubectl get jobs
kubectl get pods | grep jerry-reports
```

How many jobs and pods are currently running? Check the CronJob spec:

```bash
kubectl get cronjob jerry-reports -o yaml | grep -A5 concurrencyPolicy
kubectl get cronjob jerry-reports -o yaml | grep -E 'History|schedule'
```

What does `concurrencyPolicy: Allow` mean for a job that takes longer than its schedule interval?
