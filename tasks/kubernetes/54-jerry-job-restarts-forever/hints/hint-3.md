# Hint 3: Recreate the Job Correctly

Delete the existing broken Job:

```bash
kubectl delete job jerry-seed
```

Create a corrected version:

```bash
kubectl apply -f - <<'EOF'
apiVersion: batch/v1
kind: Job
metadata:
  name: jerry-seed
spec:
  backoffLimit: 3
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: seed
        image: busybox:1.36
        command:
        - sh
        - -c
        - |
          echo "Seeding database..."
          sleep 3
          echo "Seeding complete"
EOF
```

Watch it complete:

```bash
kubectl get job jerry-seed -w
kubectl get pods -l job-name=jerry-seed
```

The pod should show `Completed` and stay there.
